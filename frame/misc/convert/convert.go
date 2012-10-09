package convert

import(
	"labix.org/v2/mgo/bson"
	"net/url"
)

// Cleans all bson.M s to map[string]interface{} s. Usually called on db query results.
// Will become obsolete when the mgo driver will return map[string]interface{} maps instead of bson.M ones.
func Clean(x interface{}) interface{} {
	if y, ok := x.(bson.M); ok {
		for key, val := range y {
			y[key] = Clean(val)
		}
		return (map[string]interface{})(y)
	} else if d, ok := x.(map[string]interface{}); ok {
		for key, val := range d {
			d[key] = Clean(val)
		}
		return d
	} else if z, ok := x.([]interface{}); ok {
		for i, v := range z {
			z[i] = Clean(v)
		}
	}
	return x
}

func URLValuesToMap(a url.Values) map[string]interface{} {
	r := map[string]interface{}{}
	for i, v := range a {
		vi := []interface{}{}
		for _, x := range v {
			vi = append(vi, x)
		}
		if len(vi) > 1 {
			r[i] = vi
		} else {
			r[i] = vi[0]
		}
	}
	return r
}