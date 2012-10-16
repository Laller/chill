package display

// All functions which can be called from templates reside here.

import (
	"github.com/opesun/chill/frame/context"
	"github.com/opesun/chill/frame/lang"
	"github.com/opesun/chill/frame/misc/scut"
	"github.com/opesun/jsonp"
	"github.com/opesun/numcon"
	"html/template"
	"reflect"
	"net/url"
	"strings"
	"time"
	"strconv"
	"fmt"
)

func get(dat map[string]interface{}, s ...string) interface{} {
	if len(s) > 0 {
		if len(s[0]) > 0 {
			if string(s[0][0]) == "$" {
				s[0] = s[0][1:]
			}
		}
	}
	access := strings.Join(s, ".")
	val, has := jsonp.Get(dat, access)
	if !has {
		return access
	}
	return val
}

func date(timestamp int64, format ...string) string {
	var form string
	if len(format) == 0 {
		form = "2006.01.02 15:04:05"
	} else {
		form = format[0]
	}
	t := time.Unix(timestamp, 0)
	return t.Format(form)
}

func isMap(a interface{}) bool {
	v := reflect.ValueOf(a)
	switch kind := v.Kind(); kind {
	case reflect.Map:
		return true
	}
	return false
}

func eq(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func html(s string) template.HTML {
	return template.HTML(s)
}

func nonEmpty(a interface{}) bool {
	if a == nil {
		return false
	}
	switch t := a.(type) {
	case string:
		return t != ""
	case bool:
		return t != false
	default:
		return true
	}
	return true
}

// Returns the first argument which is not nil, false or empty string.
// Returns false if none of the arguments matches that criteria.
func fallback(a ...interface{}) interface{} {
	for _, v := range a {
		if nonEmpty(v) {
			return v
		}
	}
	return false
}

func formatFloat(i interface{}, prec int) string {
	f, err := numcon.Float64(i)
	if err != nil {
		return err.Error()
	}
	return strconv.FormatFloat(f, 'f', prec, 64)
}

// For debugging purposes.
func typeOf(i interface{}) string {
	return fmt.Sprint(reflect.TypeOf(i))
}

func sameKind(a, b interface{}) bool {
	return reflect.ValueOf(a).Kind() == reflect.ValueOf(b).Kind()
}

type Form struct {
	*lang.Form
}

func (f *Form) HiddenFields() [][2]string {
	ret := [][2]string{}
	for i, v := range f.FilterFields {
		for _, x := range v {
			ret = append(ret, [2]string{i, x})
		}
	}
	return ret
}

func (f *Form) HiddenString() template.HTML {
	d := f.HiddenFields()
	ret := ""
	for _, v := range d {
		ret = ret+`<input type="hidden" name="`+v[0]+`" value="`+v[1]+`" />`
	}
	return template.HTML(ret)
}

func form(action_name string, r *lang.Route, s *lang.Sentence) *Form {
	f := lang.NewURLEncoder(r, s).Form(action_name)
	return &Form{f}
}

func _url(action_name string, r *lang.Route, s *lang.Sentence, i ...string) string {
	f := lang.NewURLEncoder(r, s)
	if len(i)%2 == 1 {
		panic("Must be even.")
	}
	inp := url.Values{}
	for x:=0;x<len(i)-1; {
		inp.Add(i[x], i[x+1])
		x = x+2
	}
	return f.UrlString(action_name, inp)
}

type counter int

func newcounter() *counter {
	v := counter(0)
	return &v
}

func (c *counter) Inc() string {		// Ugly hack, template engine needs a return value.
	*c++
	return ""
}

func (c counter) Eq(i int) bool {
	return int(c) == i
}

func (c counter) EveryX(i int) bool {
	if i == 0 {
		return false
	}
	return int(c)%i==0
}

func decodeId(s string) string {
	
}

// Works from a Get or GetSingle only.
func getSub(noun string, params ...string) []interface{} {
	
}

// We must recreate this map each time because map write is not threadsafe.
// Write will happen when a hook modifies the map (hook call is not implemented yet).
func builtins(uni *context.Uni) map[string]interface{} {
	dat := uni.Dat
	user := uni.Dat["_user"]
	ret := map[string]interface{}{
		"get": func(s ...string) interface{} {
			return get(dat, s...)
		},
		"date": date,
		"is_stranger": func() bool {
			return scut.IsStranger(user)
		},
		"logged_in": func() bool {
			return !scut.IsStranger(user)
		},
		"is_guest": func() bool {
			return scut.IsGuest(user)
		},
		"is_registered": func() bool {
			return scut.IsRegistered(user)
		},
		"is_moderator": func() bool {
			return scut.IsModerator(user)
		},
		"is_admin": func() bool {
			return scut.IsAdmin(user)
		},
		"is_map": isMap,
		"eq": eq,
		"html": html,
		"format_float": formatFloat,
		"fallback": fallback,
		"type_of":	typeOf,
		"same_kind": sameKind,
		"url": func(action_name string, i ...string) string {
			return _url(action_name, uni.R, uni.S, i...) 
		},
		"form": func(action_name string) *Form {
			return form(action_name, uni.R, uni.S)
		},
		"counter": newcounter,
	}
	return ret
}
