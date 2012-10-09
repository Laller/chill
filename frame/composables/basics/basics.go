package basics

import(
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo/bson"
)

type Basics struct {
}

func (b *Basics) Get(a iface.Filter) ([]interface{}, error) {
	return a.Find()
}

func (b *Basics) Insert(a iface.Filter, data map[string]interface{}) (bson.ObjectId, error) {
	id := bson.NewObjectId()
	data["_id"] = id
	err := a.Insert(data)
	return id, err
}

func (b *Basics) Update(a iface.Filter, data map[string]interface{}) error {
	return a.Update(data)
}

func (b *Basics) UpdateAll(a iface.Filter, data map[string]interface{}) error {
	_, err := a.UpdateAll(data)
	return err
}

func (b *Basics) Remove(a iface.Filter) error {
	return a.Remove()
}

func (b *Basics) RemoveAll(a iface.Filter) error {
	_, err := a.RemoveAll()
	return err
}