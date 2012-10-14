package set

import(
	"github.com/opesun/chill/frame/misc/convert"
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo"
)

func New(db *mgo.Database, coll string) iface.Set {
	return &Set{db, coll, 0, 0, nil}
}

type Set struct {
	db 		*mgo.Database
	coll 	string
	skip	int
	limit	int
	sort	[]string
}

func (s *Set) Skip(i int) {
	s.skip = i
}

func (s *Set) Limit(i int) {
	s.limit = i
}

func (s *Set) Sort(str ...string) {
	s.sort = str
}

func (s *Set) FindOne(q map[string]interface{}) (map[string]interface{}, error) {
	var res interface{}
	err := s.db.C(s.coll).Find(q).One(&res)
	if err != nil {
		return nil, err
	}
	return convert.Clean(res).(map[string]interface{}), nil
}

func (s *Set) Count(q map[string]interface{}) (int, error) {
	return s.db.C(s.coll).Find(q).Count()
}

func (s *Set) Find(q map[string]interface{}) ([]interface{}, error) {
	c := s.db.C(s.coll).Find(q)
	if s.skip != 0 {
		c.Skip(s.skip)
	}
	if s.limit != 0 {
		c.Limit(s.limit)
	}
	if len(s.sort) > 0 {
		c.Sort(s.sort...)
	}
	var res []interface{}
	err := c.All(&res)
	if err != nil {
		return nil, err
	}
	return convert.Clean(res).([]interface{}), nil
}

func (s *Set) Insert(d map[string]interface{}) error {
	return s.db.C(s.coll).Insert(d)
}

func (s *Set) Update(q map[string]interface{}, upd_query map[string]interface{}) error {
	return s.db.C(s.coll).Update(q, upd_query)
}

func (s *Set) UpdateAll(q map[string]interface{}, upd_query map[string]interface{}) (int, error) {
	chi, err := s.db.C(s.coll).UpdateAll(q, upd_query)
	return chi.Updated, err
}

func (s *Set) Remove(q map[string]interface{}) error {
	return s.db.C(s.coll).Remove(q)
}

func (s *Set) RemoveAll(q map[string]interface{}) (int, error) {
	chi, err := s.db.C(s.coll).RemoveAll(q)
	return chi.Removed, err
}

func (s *Set) Name() string {
	return s.coll
}

