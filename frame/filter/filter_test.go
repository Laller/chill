package filter_test

import(
	"github.com/opesun/chill/frame/filter"
	"testing"
	"labix.org/v2/mgo/bson"
)

type MockEvent struct {}

func (m MockEvent) Fire(s string, params ...interface{}) {
}

func (m MockEvent) Iterate(s string, ret_rec interface{}, params ...interface{}) {
}

type TestSet struct {
	skip 		int
	limit 		int
	sort 		[]string
	lastQuery	map[string]interface{}
	name		string
	lastData	map[string]interface{}
}

func (t *TestSet) Skip(i int) {
	t.skip = i
}

func (t *TestSet) Limit(i int) {
	t.limit = i
}

func (t *TestSet) Sort(s ...string) {
	t.sort = s
}

func (t *TestSet) Name() string {
	return t.name
}

func (t *TestSet) FindOne(q map[string]interface{}) (map[string]interface{}, error) {
	t.lastQuery = q
	return nil, nil
}

func (t *TestSet) Find(q map[string]interface{}) ([]interface{}, error) {
	t.lastQuery = q
	return nil, nil
}

func (t *TestSet) Insert(d map[string]interface{}) error {
	t.lastData = d
	return nil
}

func (t *TestSet) Update(q map[string]interface{}, d map[string]interface{}) error {
	t.lastQuery = q
	return nil
}

func (t *TestSet) UpdateAll(q map[string]interface{}, d map[string]interface{}) (int, error) {
	t.lastQuery = q
	return 0, nil
}

func (t *TestSet) Remove(q map[string]interface{}) error {
	t.lastQuery = q
	return nil
}

func (t *TestSet) RemoveAll(q map[string]interface{}) (int, error) {
	t.lastQuery = q
	return 0, nil
}

func (t *TestSet) Count(q map[string]interface{}) (int, error) {
	t.lastQuery = q
	return 0, nil
}

func TestMods(t *testing.T) {
	set := &TestSet{}
	ev := &MockEvent{}
	inp := map[string]interface{}{
		"limit": 	10,
		"skip":		3,
		"sort":		[]string{"x", "y"},
	}
	f := filter.New(set, ev, inp)
	f.Find()
	if set.limit != 10 {
		t.Fatal(set.limit)
	}
	if set.skip != 3 {
		t.Fatal(set.limit)
	}
	if len(set.sort) != 2 || set.sort[0] != "x" || set.sort[1] != "y" {
		t.Fatal(set.sort)
	}
}

// Sorting could have an effect on FindOne though... For now, we specify it as irrelevant.
func TestModsSingle(t *testing.T) {
	set := &TestSet{}
	ev := &MockEvent{}
	inp := map[string]interface{}{
		"limit": 	10,
		"skip":		3,
		"sort":		[]string{"x", "y"},
	}
	f := filter.New(set, ev, inp)
	f.FindOne()
	if set.limit != 0 {
		t.Fatal(set.limit)
	}
	if set.skip != 0 {
		t.Fatal(set.limit)
	}
	if len(set.sort) != 0 {
		t.Fatal(set.sort)
	}
}

func TestQueryIn(t *testing.T) {
	set := &TestSet{}
	ev := &MockEvent{}
	inp := map[string]interface{}{
		"key": 		[]interface{}{1, 2, 3},
		"limit": 	10,
		"skip":		3,
		"sort":		[]string{"x", "y"},
	}
	f := filter.New(set, ev, inp)
	f.FindOne()
	if len(set.lastQuery) != 1 {
		t.Fatal(set.lastQuery)
	}
	keys := set.lastQuery["key"].(map[string]interface{})["$in"].([]interface{})
	if keys[0] != 1 || keys[1] != 2 || keys[2] != 3 {
		t.Fatal(keys)
	}
}

func TestCloneQuery(t *testing.T) {
	set := &TestSet{}
	ev := &MockEvent{}
	inp := map[string]interface{}{
		"crit": 	"x",
	}
	f := filter.New(set, ev, inp)
	f1 := f.Clone()
	f1.AddQuery(map[string]interface{}{
		"another_crit":	"y",
	})
	f.Find()
	if len(set.lastQuery) != 1 {
		t.Fatal(set.lastQuery)
	}
}

func TestParents(t *testing.T) {
	set := &TestSet{}
	ev := &MockEvent{}
	inp := map[string]interface{}{
		"crit": 	"x",
	}
	f := filter.New(set, ev, inp)
	// Field referencing other collection
	fieldname := "fname"
	f.AddParents(fieldname, []bson.ObjectId{bson.NewObjectId(), bson.NewObjectId(), bson.NewObjectId()})
	f.Find()
	if len(set.lastQuery) != 2 || len(set.lastQuery[fieldname].(map[string]interface{})["$in"].([]bson.ObjectId)) != 3 {
		t.Fatal(set.lastQuery)
	}
	if len(set.lastData) != 0 {
		t.Fatal(set.lastData)
	}
	f.Insert(map[string]interface{}{
		"x":"y",
	})
	if len(set.lastData) != 2 || len(set.lastData[fieldname].([]bson.ObjectId)) != 3 {
		t.Fatal(set.lastData)
	}
}

func TestAddQuerySafety(t *testing.T) {
	set := &TestSet{}
	ev := &MockEvent{}
	inp := map[string]interface{}{
		"crit": 	"x",
	}
	f := filter.New(set, ev, inp)
	add := map[string]interface{}{
		"crit":		"y",
	}
	f.AddQuery(add)
	f.Find()
	if len(set.lastQuery) != 1 || set.lastQuery["crit"] != "x" {
		t.Fatal(set.lastQuery)
	}
}