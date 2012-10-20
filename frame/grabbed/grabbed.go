package grabbed

import(
	iface "github.com/opesun/chill/frame/interfaces"
	"labix.org/v2/mgo/bson"
)

type Grabbed struct {
	set		iface.Set
	id		bson.ObjectId
}

func New(set iface.Set, id bson.ObjectId) iface.Grabbed {
	return &Grabbed{set, id}
}

func (g *Grabbed) Update(upd map[string]interface{}) error {
	q := map[string]interface{}{
		"_id": 	g.id,
	}
	return g.set.Update(q, upd)
}

func (g *Grabbed) Remove() error {
	q := map[string]interface{}{
		"_id": 	g.id,
	}
	return g.set.Remove(q)
}