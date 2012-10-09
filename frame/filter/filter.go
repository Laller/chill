package filter

import(
	"fmt"
	"github.com/opesun/chill/frame/set"
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo/bson"
	"encoding/base64"
)

type mods struct {
	skip		int
	limit		int
	page		int
	perPage		int
	sort		string
}

type Filter struct {
	set				set.SetInterface
	mods			*mods
	parentField		string
	parents			map[string][]bson.ObjectId
	query			map[string]interface{}
}

func (f *Filter) Visualize() {
	fmt.Println("<<<")
	fmt.Println("fmod", f.mods)
	fmt.Println("parents", f.parents)
	fmt.Println("query", f.query)
	fmt.Println(">>>")
}

func (f *Filter) Reduce(a ...iface.Filter) (iface.Filter, error) {
	l := len(a)
	if l == 0 {
		return &Filter{}, fmt.Errorf("Nothing to reduce.")
	}
	var prev iface.Filter
	prev = f
	for _, v := range a {
		ids, err := prev.Ids()
		if err != nil {
			return &Filter{}, err
		}
		v.AddParents(prev.Subject(), ids)
		prev = v
	}
	return prev, nil
}

// Information coming from url.Values/map
type data struct {
	query 		map[string]interface{}
	mods		*mods
	parentField	string
}

func processMap(inp map[string]interface{}) *data {
	d := &data{}
	if inp == nil {
		return d
	}
	if val, has := inp["parentf"]; has {
		d.parentField = val.(string)
		delete(inp, "parentf")
	}
	mods := &mods{}
	if val, has := inp["sort"]; has {
		mods.sort = val.(string)
		delete(inp, "sort")
	}
	d.mods = mods
	d.query = toQuery(inp)
	return d
}

func decodeId(s string) bson.ObjectId {
	val, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		panic("Can't decode id: "+ err.Error())
	}
	return bson.ObjectId(val)
}

// inp: url.Values => map, returns query map
func toQuery(a map[string]interface{}) map[string]interface{} {
	r := map[string]interface{}{}
	for i, v := range a {
		var vi []interface{}
		if slice, ok := v.([]interface{}); ok {
			for _, x := range slice {
				if i == "id" {
					i = "_id"
					vi = append(vi, decodeId(v.(string)))
				} else {
					vi = append(vi, x)
				}
			}
		} else {
			if i == "id" {
				i = "_id"
				vi = append(vi, decodeId(v.(string)))
			} else {
				vi = append(vi, v)
			}
		}
		if len(vi) > 1 {
			r[i] = map[string]interface{}{
				"$in": vi,
			}
		} else {
			r[i] = vi[0]
		}
	}
	return r
}

func NewSimple(set set.SetInterface) *Filter {
	return &Filter{
		set:			set,
		parents:		map[string][]bson.ObjectId{},
	}
}

func New(set set.SetInterface, all map[string]interface{}) *Filter {
	d := processMap(all)
	f := &Filter{
		set:			set,
		mods:			d.mods,
		parentField:	d.parentField,
		query:			d.query,
		parents:		map[string][]bson.ObjectId{},
	}
	return f
}

func (f *Filter) Clone() iface.Filter {
	//panic("Clone is not implemented yet.")
	//q := copyMap()
	//p := copySlice()
	//return &Filter{
	//	set:			f.set,
	//	mods:			&*d.mods,
	//	parentField:	d.ParentField,
	//	query:			d.query
	//	parents:		d.parents
	//}
	return f
}

func (f *Filter) AddQuery(q map[string]interface{}) iface.Filter {
	for i, v := range f.query {
		q[i] = v
	}
	f.query = q
	return f
}

func mergeQuery(q map[string]interface{}, p map[string][]bson.ObjectId) map[string]interface{} {
	r := map[string]interface{}{}
	for i, v := range q {
		r[i] = v
	}
	for i, v := range p {
		r[i] = map[string]interface{}{
			"$in": v,
		}
	}
	return r
}

func mergeInsert(ins map[string]interface{}, p map[string][]bson.ObjectId) map[string]interface{} {
	r := map[string]interface{}{}
	for i, v := range ins {
		r[i] = v
	}
	for i, v := range p {
		r[i] = v
	}
	return r
}

func (f *Filter) AddCrit(q map[string]interface{}) {
}

func (f *Filter) FindOne() (map[string]interface{}, error) {
	q := mergeQuery(f.query, f.parents)
	return f.set.FindOne(q)
}

func (f *Filter) Find() ([]interface{}, error) {
	q := mergeQuery(f.query, f.parents)
	return f.set.Find(q)
}

func (f *Filter) Insert(d map[string]interface{}) error {
	i := mergeInsert(d, f.parents)
	return f.set.Insert(i)
}

func (f *Filter) Update(upd_query map[string]interface{}) error {
	q := mergeQuery(f.query, f.parents)
	return f.set.Update(q, upd_query)
}

func (f *Filter) UpdateAll(upd_query map[string]interface{}) (int, error) {
	q := mergeQuery(f.query, f.parents)
	return f.set.UpdateAll(q, upd_query)
}

func (f *Filter) Subject() string {
	return f.set.Name()
}

func (f *Filter) AddParents(fieldname string, a []bson.ObjectId) {
	if len(f.parentField) > 0 {
		fieldname = f.parentField
	} else {
		fieldname = "_"+fieldname
	}
	slice, ok := f.parents[fieldname]
	if !ok {
		f.parents[fieldname] = []bson.ObjectId{}
		slice = []bson.ObjectId{}
	}
	slice = append(slice, a...)
	f.parents[fieldname] = slice
}

func (f *Filter) Ids() ([]bson.ObjectId, error) {
	if val, has := f.query["id"]; has && len(f.query) == 1 && len(f.parents) == 1 {
		ids := val.(map[string]interface{})["$in"].([]interface{})
		ret := []bson.ObjectId{}
		for _, v := range ids {
			ret = append(ret, v.(bson.ObjectId))
		}
		return ret, nil
	}
	q := mergeQuery(f.query, f.parents)
	docs, err := f.set.Find(q)
	if err != nil {
		return nil, err
	}
	ret := []bson.ObjectId{}
	for _, v := range docs {
		ret = append(ret, v.(map[string]interface{})["_id"].(bson.ObjectId))
	}
	return ret, nil
}

func (f *Filter) Remove() error {
	q := mergeQuery(f.query, f.parents)
	return f.set.Remove(q)
}

func (f *Filter) RemoveAll() (int, error) {
	q := mergeQuery(f.query, f.parents)
	return f.set.RemoveAll(q)
}