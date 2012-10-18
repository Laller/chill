package display

// All functions which can be called from templates reside here.

import (
	"github.com/opesun/chill/frame/context"
	"github.com/opesun/chill/frame/lang"
	"github.com/opesun/chill/frame/glue"
	"github.com/opesun/chill/frame/misc/convert"
	"github.com/opesun/chill/frame/misc/scut"
	"github.com/opesun/jsonp"
	"github.com/opesun/paging"
	"labix.org/v2/mgo/bson"
	"github.com/opesun/numcon"
	"html/template"
	"reflect"
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
	return ""
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
		if _, yepp := v.([]interface{}); yepp {
			for _, x := range v.([]interface{}) {
				ret = append(ret, [2]string{i, fmt.Sprint(x)})
			}
		} else {
			ret = append(ret, [2]string{i, fmt.Sprint(v)})
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

func _url(action_name string, r *lang.Route, s *lang.Sentence, i ...interface{}) string {
	f := lang.NewURLEncoder(r, s)
	if len(i)%2 == 1 {
		panic("Must be even.")
	}
	inp := convert.ListToMap(i...)
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

// Mainly designed to work from Get or GetSingle
func getSub(uni *context.Uni, noun string, params ...interface{}) []interface{} {
	if uni.Route == nil && uni.Sentence == nil {
		panic("Nothing to do here.")
	}
	s := uni.Sentence
	r := uni.Route
	var path string
	var urls []map[string]interface{}
	if s.Verb != "Get" && s.Verb != "GetSingle" {
		path = "/" + strings.Join(r.Words, "/") + "/" + noun
		urls = append(urls, r.Queries...)
		urls = append(urls, convert.ListToMap(params...))
	} else {
		path = "/" + strings.Join(r.Words, "/") + "/" + noun
		urls = append(urls, r.Queries[:len(r.Queries)-1]...)
		urls = append(urls, convert.ListToMap(params...))
	}
	desc, err := glue.Identify(path, uni.Opt["nouns"].(map[string]interface{}), lang.EncodeQueries(urls, false))
	inp, data, err := desc.CreateInputs(uni.FilterCreator)
	if err != nil {
		panic(err)
	}
	if data != nil {
		inp = append(inp, data)
	}
	module := uni.NewModule(desc.VerbLocation)
	if !module.Exists() {
		panic("Module does not exist.")
	}
	ins := module.Instance()
	ret := []interface{}{}
	ret_rec := func(i ...interface{}) {
		ret = i
	}
	ins.Method(uni.Sentence.Verb).Call(ret_rec, inp...)
	return ret
}

func getList(uni *context.Uni, noun string, params ...interface{}) []interface{} {
	values := convert.ListToMap(params...)
	desc, err := glue.Identify("/"+noun, uni.Opt["nouns"].(map[string]interface{}), values)
	inp, data, err := desc.CreateInputs(uni.FilterCreator)
	if err != nil {

		panic(err)
	}
	if data != nil {
		inp = append(inp, data)
	}
	module := uni.NewModule(desc.VerbLocation)
	if !module.Exists() {
		panic("Module does not exist.")
	}
	ins := module.Instance()
	ret := []interface{}{}
	ret_rec := func(i ...interface{}) {
		ret = i
	}
	ins.Method("Get").Call(ret_rec, inp...)
	return ret
}

func decodeId(s string) string {
	val := convert.DecodeIdP(s)
	return val.Hex()
}

func hexId(a bson.ObjectId) string {
	return a.Hex()
}

type pagr struct {
	HasPrev		bool
	Prev		int
	HasNext		bool
	Next		int
	Elems		[]paging.Pelem
}

func pager(uni *context.Uni, pagestr string, count, limit int) []paging.Pelem {
	if len(pagestr) == 0 {
		pagestr = "1"
	}
	if limit == 0 {
		return nil
	}
	p := uni.Path + "?" + uni.Req.URL.RawQuery
	page, err := strconv.Atoi(pagestr)
	if err != nil {
		return nil	// Not blowing up here.
	}
	if page == 0 {
		return nil
	}
	page_count := count/limit+1
	nav, _ := paging.P(page, page_count, 3, p)
	return nav
}

func elem(s []interface{}, memb int) interface{} {
	return s[memb]
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
		"title": strings.Title,
		"url": func(action_name string, i ...interface{}) string {
			return _url(action_name, uni.Route, uni.Sentence, i...) 
		},
		"form": func(action_name string) *Form {
			return form(action_name, uni.Route, uni.Sentence)
		},
		"counter": newcounter,
		"get_sub": func(s string, params ...interface{}) []interface{} {
			return getSub(uni, s, params...)
		},
		"get_list": func(s string, params ...interface{}) []interface{} {
			return getList(uni, s, params...)
		},
		"decode_id": decodeId,
		"hex_id": hexId,
		"elem": elem,
		"pager": func(pagesl []string, count, limited int) []paging.Pelem {
			var pagestr string
			if len(pagesl) == 0 {
				pagestr = "1"
			} else {
				pagestr = pagesl[0]
			}
			return pager(uni, pagestr, count, limited)
		},
	}
	uni.Ev.Fire("AddTemplateBuiltin", ret)
	return ret
}
