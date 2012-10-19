package filter

import(
	"fmt"
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo/bson"
	"github.com/opesun/chill/frame/misc/convert"
	"github.com/opesun/sanitize"
)

type Mods struct {
	skip		int
	limit		int
	sort		[]string
}

func (m *Mods)  Skip() int {
	return m.skip
}

func (m *Mods) Limit() int {
	return m.limit
}

func (m *Mods) Sort() []string {
	return m.sort
}

type Filter struct {
	set				iface.Set
	mods			*Mods
	parentField		string
	parents			map[string][]bson.ObjectId
	query			map[string]interface{}
	ev				iface.Event
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
	mods		*Mods
	parentField	string
}

// Special fields in query:
// parentf, sort, limit, skip, page
func processMap(inp map[string]interface{}, ev iface.Event) *data {
	d := &data{}
	if inp == nil {
		inp = map[string]interface{}{}
	}
	int_sch := map[string]interface{}{
		"type": "int",
	}
	sch := map[string]interface{}{
		"parentf": 1,
		"sort": map[string]interface{}{
			"slice": true,
			"type": "string",
		},
		"skip": int_sch,
		"limit": int_sch,
		"page": int_sch,
	}
	ex, err := sanitize.New(sch)
	if err != nil {
		panic(err)
	}
	dat, err := ex.Extract(inp)
	if err != nil {
		panic(err)
	}
	for i := range sch {
		delete(inp, i)
	}
	mods := &Mods{}
	if dat["parentf"] != nil {
		d.parentField = dat["parentf"].(string)
	}
	if dat["sort"] != nil {
		mods.sort = convert.ToStringSlice(dat["sort"].([]interface{}))
	}
	if dat["skip"] != nil {
		mods.skip = int(dat["skip"].(int64))
	}
	if dat["limit"] != nil {
		mods.limit = int(dat["limit"].(int64))
	} else {
		mods.limit = 20
	}
	if dat["page"] != nil {
		page := int(dat["page"].(int64))
		mods.skip = (page-1)*mods.limit
	}
	d.mods = mods
	ev.Fire("ProcessMap", inp)	// We should let the subscriber now the subject name.
	d.query = toQuery(inp)
	return d
}

func convAppend(vi []interface{}, i string, x interface{}) []interface{} {
	if i == "id" {
		i = "_id"
		vi = append(vi, convert.DecodeIdP(x.(string)))
	} else {
		vi = append(vi, x)
	}
	return vi
}

// map => mongodb query map
func toQuery(a map[string]interface{}) map[string]interface{} {
	r := map[string]interface{}{}
	for i, v := range a {
		var vi []interface{}
		if slice, ok := v.([]interface{}); ok {
			for _, x := range slice {
				vi = convAppend(vi, i, x)
			}
		} else {
			vi = convAppend(vi, i, v)
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

func NewSimple(set iface.Set, ev iface.Event) *Filter {
	return &Filter{
		set:			set,
		parents:		map[string][]bson.ObjectId{},
		ev:				ev,
	}
}

func New(set iface.Set, all map[string]interface{}, ev iface.Event) *Filter {
	d := processMap(all, ev)
	f := &Filter{
		set:			set,
		mods:			d.mods,
		parentField:	d.parentField,
		query:			d.query,
		parents:		map[string][]bson.ObjectId{},
		ev:				ev,
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

func (f *Filter) Modifiers() iface.Modifiers {
	return f.mods
}

func (f *Filter) AddQuery(q map[string]interface{}) iface.Filter {
	query := processMap(q, f.ev).query
	for i, v := range f.query {
		query[i] = v
	}
	f.query = query
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

func (f *Filter) FindOne() (map[string]interface{}, error) {
	q := mergeQuery(f.query, f.parents)
	return f.set.FindOne(q)
}

func (f *Filter) Find() ([]interface{}, error) {
	q := mergeQuery(f.query, f.parents)
	if f.mods.skip != 0 {
		f.set.Skip(f.mods.skip)
	}
	if f.mods.limit != 0 {
		f.set.Limit(f.mods.limit)
	}
	if len(f.mods.sort) > 0 {
		f.set.Sort(f.mods.sort...)
	}
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

func (f *Filter) Count() (int, error) {
	q := mergeQuery(f.query, f.parents)
	return f.set.Count(q)
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