// Context contains the type Uni. An instance of this type is passed to the modules when routing the control to them.
package context

import (
	"github.com/opesun/chill/frame/misc/convert"
	"github.com/opesun/chill/frame/lang"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/jsonp"
	"labix.org/v2/mgo"
	"net/http"
	"strings"
	"reflect"
	"fmt"
)

// General context for the application.
type Uni struct {
	Modifiers			map[string]interface{}
	Session 			*mgo.Session
	Db      			*mgo.Database
	W       			http.ResponseWriter
	Req     			*http.Request
	secret  			string                 		// Used for things like encryption/decryption. Basically a permanent random data.
	Path       			string                 		// Path string
	opt     			string                 		// Original string representation of the option, if one needs a version which is guaranteedly untinkered.
	Opt     			map[string]interface{} 		// Freshest options from database.
	Dat     			map[string]interface{} 		// General communication channel.
	Put     			func(...interface{})   		// Just a convenience function to allow fast output to http response.
	Root    			string                 		// Absolute path of the application.
	Ev      			*Ev
	Route				*lang.Route
	Sentence			*lang.Sentence
	FilterCreator		func(string, map[string]interface{}) iface.Filter
	NewModule			func(string) iface.Module
}

// Set only once.
func (u *Uni) SetOriginalOpt(s string) {
	if u.opt == "" {
		u.opt = s
	}
}

func (u *Uni) OriginalOpt() string {
	return u.opt
}

// Maybe we should not even return the secret, because a badly written module can make it public.
// Or, we could serve different values to different packages.
// That makes the encrypted values noncompatible across packages though.
func (u *Uni) Secret() string {
	return u.secret
}

// Set only once.
func (u *Uni) SetSecret(s string) {
	if u.secret == "" {
		u.secret = s
	}
}

// Used to call subscribed hooks.
type Ev struct {
	uni    		*Uni
	cache		map[string]iface.Instance
	newModule	func(string) iface.Module
}

// Return all hooks modules subscribed to a path.
func all(e *Ev, path string) []string {
	modnames, ok := jsonp.GetS(e.uni.Opt, "Hooks." + path)
	if !ok {
		return nil
	}
	ret := []string{}
	for _, v := range modnames {
		ret = append(ret, v.(string))
	}
	return ret
}

// This is an iface.Module, which wraps the github.com/opesun/chill/frame/mod implementation and implements instance caching.
type InstanceCacher struct {
				iface.Module
	cache 		map[string]iface.Instance
	uni			*Uni
	name		string
}

func (m InstanceCacher) Instance() iface.Instance {
	var ins iface.Instance
	ins, has := m.cache[m.name]
	if !has {
		if !m.Exists() {
			panic(fmt.Sprintf("Module %v modname does not exist.", m.name))
		}
		insta := m.Module.Instance()
		insta.Method("Init").Call(nil, m.uni)
		m.cache[m.name] = insta
		ins = insta
	}
	return ins
}

func (e *Ev) NewModuleProducer() func(string) iface.Module {
	return func(modname string) iface.Module {
		return &InstanceCacher{
			e.newModule(modname),
			e.cache,
			e.uni,
			modname,
		}
	}
}

// Fire calls hooks subscribed to eventname, but does not case about their return values.
func (e *Ev) Fire(eventname string, params ...interface{}) {
	e.iterate(eventname, nil, params...)
}

// Calls all hooks subscribed to eventname, with params, feeding the output of every hook into stopfunc.
// Stopfunc's argument signature must match the signatures of return values of the called hooks.
// Stopfunc must return a boolean value. A boolean value of true stops the iteration.
// Iterate allows to mimic the semantics of calling all hooks one by one.
func (e *Ev) Iterate(eventname string, stopfunc interface{}, params ...interface{}) {
	e.iterate(eventname, stopfunc, params...)
}

func (e *Ev) iterate(eventname string, stopfunc interface{}, params ...interface{}) {
	subscribed := all(e, eventname)
	hookname := hooknameize(eventname)
	var stopfunc_numin int
	if stopfunc != nil {
		s := reflect.TypeOf(stopfunc)
		if s.Kind() != reflect.Func {
			panic("Stopfunc is not a function.")
		}
		if s.NumOut() != 1 {
			panic("Stopfunc must have one return value.")
		}
		if s.Out(0) != reflect.TypeOf(false) {
			panic("Stopfunc must have a boolean return value.")
		}
		stopfunc_numin = s.NumIn()
	}
	for _, modname := range subscribed {
		hook_outp := []reflect.Value{}
		var ins iface.Instance
		ins, has := e.cache[modname]
		if !has {
			mo := e.newModule(modname)
			if !mo.Exists() {
				panic(fmt.Sprintf("Module %v modname does not exist.", modname))
			}
			insta := mo.Instance()
			insta.Method("Init").Call(nil, e.uni)
			e.cache[modname] = insta
			ins = insta
		}
		if !ins.HasMethod(hookname) {
			panic(fmt.Sprintf("Module %v has no method named %v", modname, hookname))
		}
		var ret_rec interface{}
		if stopfunc != nil {
			ret_rec = func(i ...interface{}) {
				for i, v := range i {
					if v == nil {
						hook_outp = append(hook_outp, reflect.Zero(reflect.TypeOf(stopfunc).In(i)))
					} else {
						hook_outp = append(hook_outp, reflect.ValueOf(v))
					}
				}
			}
		}
		err := ins.Method(hookname).Call(ret_rec, params...)
		if err != nil {
			panic(err)
		}
		if stopfunc != nil {
			if stopfunc_numin != len(hook_outp) {
				panic(fmt.Sprintf("The number of return values of Hook %v of %v differs from the number of arguments of stopfunc.", hookname, modname))	// This sentence...
			}
			stopf := reflect.ValueOf(stopfunc)
			stopf_ret := stopf.Call(hook_outp)
			if stopf_ret[0].Interface().(bool) == true {
				break
			}
		}
	}
}

func NewEv(uni *Uni, newModule func(string)iface.Module) *Ev {
	return &Ev{uni, map[string]iface.Instance{}, newModule}
}

// Creates a hookname from access path.
// "content.insert" => "ContentInsert"
func hooknameize(s string) string {
	s = strings.Replace(s, ".", " ", -1)
	s = strings.Title(s)
	return strings.Replace(s, " ", "", -1)
}

var Convert = convert.Clean