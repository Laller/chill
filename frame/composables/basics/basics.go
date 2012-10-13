package basics

import(
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo/bson"
	"fmt"
)

type Basics struct {
}

func (b *Basics) Get(a iface.Filter) ([]interface{}, error) {
	return a.Find()
}

func (b *Basics) Insert(a iface.Filter, data map[string]interface{}) (bson.ObjectId, error) {
	fmt.Println("subb:", a.Subject())
	id := bson.NewObjectId()
	data["_id"] = id
	err := a.Insert(data)
	return id, err
}

func (b *Basics) Update(a iface.Filter, data map[string]interface{}) error {
	upd := map[string]interface{}{
		"$set": data,
	}
	return a.Update(upd)
}

func (b *Basics) UpdateAll(a iface.Filter, data map[string]interface{}) error {
	upd := map[string]interface{}{
		"$set": data,
	}
	_, err := a.UpdateAll(upd)
	return err
}

func (b *Basics) Remove(a iface.Filter) error {
	return a.Remove()
}

func (b *Basics) RemoveAll(a iface.Filter) error {
	_, err := a.RemoveAll()
	return err
}