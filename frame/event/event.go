package event

import(
	"github.com/opesun/jsonp"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/context"
	"strings"
	"reflect"
	"fmt"
)

// Used to call subscribed hooks.
type Ev struct {
	uni    		*context.Uni
	cache		map[string]iface.Instance
	newModule	func(string) iface.Module
}

func NewEv(uni *context.Uni, newModule func(string)iface.Module) *Ev {
	return &Ev{
		uni,
		map[string]iface.Instance{},
		newModule,
	}
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
	uni			*context.Uni
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
		if insta.HasMethod("Init") {
			insta.Method("Init").Call(nil, m.uni)
		}
		m.cache[m.name] = insta
		return insta
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

func validateStopFunc(s reflect.Type) error {
	if s.Kind() != reflect.Func {
		return fmt.Errorf("Stopfunc is not a function.")
	}
	if s.NumOut() != 1 {
		return fmt.Errorf("Stopfunc must have one return value.")
	}
	if s.Out(0) != reflect.TypeOf(false) {
		return fmt.Errorf("Stopfunc must have a boolean return value.")
	}
	return nil
}

func (e *Ev) instance(modname string) iface.Instance {
	ins, exists := e.cache[modname]
	if exists {
		return ins
	}
	mo := e.newModule(modname)
	if !mo.Exists() {
		panic(fmt.Sprintf("Module %v modname does not exist.", modname))
	}
	insta := mo.Instance()
	if insta.HasMethod("Init") {
		insta.Method("Init").Call(nil, e.uni)
	}
	e.cache[modname] = insta
	return insta
}

func (e *Ev) iterate(eventname string, stopfunc interface{}, params ...interface{}) {
	subscribed := all(e, eventname)
	hookname := hooknameize(eventname)
	var stopfunc_numin int
	if stopfunc != nil {
		s := reflect.TypeOf(stopfunc)
		err := validateStopFunc(s)
		if err != nil {
			panic(err)
		}
		stopfunc_numin = s.NumIn()
	}
	for _, modname := range subscribed {
		hook_outp := []reflect.Value{}
		ins := e.instance(modname)
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

// Creates a hookname from access path.
// "content.insert" => "ContentInsert"
func hooknameize(s string) string {
	s = strings.Replace(s, ".", " ", -1)
	s = strings.Title(s)
	return strings.Replace(s, " ", "", -1)
}