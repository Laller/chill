package basics

import(
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo/bson"
)

type Basics struct {
	Ev iface.Event		// Not entirely sure if the triggering of Events should be placed here, we should probably move it above this layer.
}

type QueryInfo struct {
	Count 	int
	Skipped	int
	Limited	int
	Sorted	[]string
}

func (b *Basics) Get(a iface.Filter) ([]interface{}, *QueryInfo, error) {
	list, err := a.Find()
	if err != nil {
		return nil, nil, err
	}
	count, err := a.Count()
	if err != nil {
		return nil, nil, err
	}
	return list, &QueryInfo{
		count, a.Modifiers().Skip(),
		a.Modifiers().Limit(),
		a.Modifiers().Sort(),
	}, nil
}

func (b *Basics) GetSingle(a iface.Filter) (map[string]interface{}, error) {
	return a.FindOne()
}

func (b *Basics) Insert(a iface.Filter, data map[string]interface{}) (bson.ObjectId, error) {
	id := bson.NewObjectId()
	data["_id"] = id
	err := a.Insert(data)
	if err != nil {
		return "", err
	}
	if b.Ev != nil {
		q := map[string]interface{}{
			"_id": id,
		}
		filt := a.Clone().AddQuery(q)
		b.Ev.Fire("Inserted", filt)
		b.Ev.Fire(a.Subject() + "Inserted", filt)
	}
	return id, nil
}

func (b *Basics) Update(a iface.Filter, data map[string]interface{}) error {
	upd := map[string]interface{}{
		"$set": data,
	}
	err := a.Update(upd)
	if err != nil {
		return err
	}
	if b.Ev != nil {
		b.Ev.Fire("Updated", a)
		b.Ev.Fire(a.Subject() + "Updated", a)
	}
	return nil
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