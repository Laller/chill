package interfaces

import (
	"reflect"
	"labix.org/v2/mgo/bson"
)

type Event interface {
	Fire(eventname string, params ...interface{})
	Iterate(eventname string, stopfunc interface{}, params ...interface{})
}

type Method interface {
	Call(interface{}, ...interface{}) error
	Matches(interface{}) bool
	InputTypes() []reflect.Type
	OutputTypes() []reflect.Type
}

type Instance interface {
	HasMethod(string) bool
	MethodNames() []string
	Method(string) Method
}

type Module interface {
	Instance() Instance
	Exists() bool
}

type Speaker interface {
	IsNoun(string) bool
	NounHasVerb(string, string) bool
}

type Set interface {
	Skip(int)
	Limit(int)
	Sort(...string)
	Count(map[string]interface{}) (int, error)
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

type Filter interface {
	Ids() ([]bson.ObjectId, error)
	AddQuery(map[string]interface{}) Filter
	Clone() Filter
	Reduce(...Filter) (Filter, error)
	Subject() string
	AddParents(string, []bson.ObjectId)
	Modifiers() Modifiers
	Count()	(int, error)
	// --
	FindOne() (map[string]interface{}, error)
	Find() ([]interface{}, error)
	Insert(map[string]interface{}) error
	// InsertAll([]map[string]interface{}) errors
	Update(map[string]interface{}) error
	UpdateAll(map[string]interface{}) (int, error)
	Remove() error
	RemoveAll() (int, error)
}

type Modifiers interface {
	Sort()		[]string
	Limit()		int
	Skip()		int
}