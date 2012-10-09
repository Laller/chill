package set

import(
	"github.com/opesun/chill/frame/misc/convert"
	"labix.org/v2/mgo"
)

type SetInterface interface {
	FindOne(map[string]interface{}) (map[string]interface{}, error)
	Find(map[string]interface{}) ([]interface{}, error)
	Insert(map[string]interface{}) error
	// InsertAll([]map[string]interface{}) errors
	Update(map[string]interface{}, map[string]interface{}) error
	UpdateAll(map[string]interface{}, map[string]interface{}) (int, error)
	Remove(map[string]interface{}) error
	RemoveAll(map[string]interface{}) (int, error)
	Name()	string
}

func New(db *mgo.Database, coll string) SetInterface {
	return &Set{db, coll}
}

type Set struct {
	db *mgo.Database
	coll string
}

func (s *Set) FindOne(q map[string]interface{}) (map[string]interface{}, error) {
	var res interface{}
	err := s.db.C(s.coll).Find(q).One(&res)
	if err != nil {
		return nil, err
	}
	return convert.Clean(res).(map[string]interface{}), nil
}

func (s *Set) Find(q map[string]interface{}) ([]interface{}, error) {
	var res []interface{}
	err := s.db.C(s.coll).Find(q).All(&res)
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

