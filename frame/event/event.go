package event

import(
	"github.com/opesun/jsonp"
	iface "github.com/opesun/chill/frame/interfaces"
	"strings"
	"reflect"
	"fmt"
)

// Used to call subscribed hooks.
type Ev struct {
	hooks		map[string]interface{}
	passOn   	interface{}						// We pass this on to the Init method of the module instances. In reality, this is a *context.Uni.
	cache		map[string]iface.Instance		// Module instance cache.
	newModule	func(string) iface.Module
}

func New(pass_on interface{}, hooks map[string]interface{}, newModule func(string)iface.Module) *Ev {
	if hooks == nil {
		hooks = map[string]interface{}{}
	}
	return &Ev{
		hooks,
		pass_on,
		map[string]iface.Instance{},
		newModule,
	}
}

type hookInf struct {
	modName			string
	methodName		string
}

// Return all hooks modules subscribed to a path.
func all(e *Ev, path string) []hookInf {
	ret := []hookInf{}
	subscribed, ok := jsonp.GetS(e.hooks, path)
	if !ok {
		return ret
	}
	for _, v := range subscribed {
		hinf := hookInf{}
		switch t := v.(type) {
		case string:
			hinf.modName = t
		case []interface{}:
			if len(t) != 2 {
				panic("Misconfigured hook.")
			}
			hinf.modName = t[0].(string)
			hinf.methodName = t[1].(string)
		}
		ret = append(ret, hinf)
	}
	return ret
}

// This is an iface.Module, which wraps the github.com/opesun/chill/frame/mod implementation and implements instance caching.
type InstanceCacher struct {
				iface.Module
	cache 		map[string]iface.Instance
	passOn		interface{}
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
			insta.Method("Init").Call(nil, m.passOn)
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
			e.passOn,
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
		panic(fmt.Sprintf("Module %v does not exist.", modname))
	}
	insta := mo.Instance()
	if insta.HasMethod("Init") {
		insta.Method("Init").Call(nil, e.passOn)
	}
	e.cache[modname] = insta
	return insta
}

func (e *Ev) iterate(eventname string, stopfunc interface{}, params ...interface{}) {
	subscribed := all(e, eventname)
	var stopfunc_numin int
	if stopfunc != nil {
		s := reflect.TypeOf(stopfunc)
		err := validateStopFunc(s)
		if err != nil {
			panic(err)
		}
		stopfunc_numin = s.NumIn()
	}
	nameized := hooknameize(eventname)
	for _, hinf := range subscribed {
		if hinf.methodName == "" {
			hinf.methodName = nameized
		}
		ins := e.instance(hinf.modName)
		if !ins.HasMethod(hinf.methodName) {
			panic(fmt.Sprintf("Module %v has no method named %v", hinf.modName, hinf.methodName))
		}
		var ret_rec interface{}
		hook_outp := []reflect.Value{}
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
		err := ins.Method(hinf.methodName).Call(ret_rec, params...)
		if err != nil {
			panic(err)
		}
		if stopfunc != nil {
			if stopfunc_numin != len(hook_outp) {
				panic(fmt.Sprintf("The number of return values of Hook %v of %v differs from the number of arguments of stopfunc.", hinf.methodName, hinf.modName))	// This sentence...
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