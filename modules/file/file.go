package file

import(
	"github.com/opesun/chill/frame/context"
	"github.com/opesun/jsonp"
	"github.com/opesun/chill/frame/misc/convert"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/composables/basics"
	"labix.org/v2/mgo/bson"
	"fmt"
	"github.com/opesun/sanitize"
	"strings"
	"mime/multipart"
	"io/ioutil"
	"path/filepath"
	"bytes"
	"os"
)

type C struct {
	basics.Basics
	uni *context.Uni
	fileBiz map[string]interface{}
}

func (c *C) Init(uni *context.Uni) {
	c.uni = uni
	c.fileBiz = map[string]interface{}{} 
}

func (c *C) SanitizerMangler(san *sanitize.Extractor) {
	san.AddFuncs(sanitize.FuncMap{
		"file": func(dat interface{}, s sanitize.Scheme) (interface{}, error) {
			if c.uni.Req.MultipartForm.File == nil {
				return nil, fmt.Errorf("No files at all.")
			}
			val, has := c.uni.Req.MultipartForm.File[s.Key]
			if !has {
				return nil, fmt.Errorf("Can't find key amongst files.")
			}
			ret := []interface{}{}
			for _, v := range val {
				ret = append(ret, v)
			}
			c.fileBiz[s.Key] = ret
			return ret, nil
		},
	})
}

func (c *C) SanitizedDataMangler(data map[string]interface{}) {
	if len(c.fileBiz) > 0 {
		data["_files"] = c.fileBiz
	}
}

func copy(fh *multipart.FileHeader, path, fname string) error {
	buf := new(bytes.Buffer)
	file, err := fh.Open()
	if err != nil {
		return err
	}
	_, err = buf.ReadFrom(file)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path, 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(path, fname), buf.Bytes(), os.FileMode(0644))
}

func fname(fullp string) string {
	s := strings.Split(fullp, "/")
	return s[len(s)-1]
}

func sanitizeHost(host string) string {
	return strings.Replace(host, ":", "-", -1)
}

func (c *C) moveFiles(subject, id string, files map[string]interface{}) error {
	to := filepath.Join(c.uni.Root, "uploads", sanitizeHost(c.uni.Req.Host), subject, id)
	for folder, slice := range files {
		for _, fh_i := range slice.([]interface{}) {
			fh := fh_i.(*multipart.FileHeader)
			fname := fh.Filename
			err := copy(fh, filepath.Join(to, folder), fname)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Converts all absolute paths to filenames in the files map.
func fileheadersToFilenames(files map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}
	for folder, slice := range files {
		if _, has := ret[folder]; !has {
			ret[folder] = []interface{}{}
		}
		for _, fh_i := range slice.([]interface{}) {
			fh := fh_i.(*multipart.FileHeader)
			ret[folder] = append(ret[folder].([]interface{}), fh.Filename)
		}
	}
	return ret
}

func merge(a, b map[string]interface{}) map[string]interface{} {
	for i, v := range b {
		a[i] = v
	}
	return a
}

func (c *C) Insert(a iface.Filter, data map[string]interface{}) (bson.ObjectId, error) {
	files_map, has_files := data["_files"].(map[string]interface{})
	if has_files {
		delete(data, "_files")
		merge(data, fileheadersToFilenames(files_map))
	}
	id, err := c.Basics.Insert(a, data)
	if err != nil {
		return id, err
	}
	if has_files {
		err := c.moveFiles(a.Subject(), id.Hex(), files_map)
		if err != nil {
			return "", err
		}
	}
	return id, nil
}

func eachIfNeeded(filenames map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}
	for folder, sl := range filenames {
		slice := sl.([]interface{})
		l := len(slice)
		if l == 0 {
			panic("WTF")
		} else if l == 1 {
			ret[folder] = slice[0]
		} else {
			each := []interface{}{}
			for _, v := range slice {
				each = append(each, v)
			}
			ret[folder] = map[string]interface{}{
				"$each": each,
			}
		}
	}
	return ret
}

func (c *C) Update(a iface.Filter, data map[string]interface{}) error {
	files_map, has := data["_files"].(map[string]interface{})
	upd := map[string]interface{}{
		"$set": data,
	}
	if has {
		ids, err := a.Ids()
		if err != nil {
			return err
		}
		err = c.moveFiles(a.Subject(), ids[0].Hex(), files_map)
		if err != nil {
			return err
		}
		delete(data, "_files")
		upd["$addToSet"] = eachIfNeeded(fileheadersToFilenames(files_map))
	}
	return a.Update(upd)
}

func (c *C) getScheme(noun, verb string) (map[string]interface{}, error) {
	scheme, ok := jsonp.GetM(c.uni.Opt, fmt.Sprintf("nouns.%v.verbs.%v.input", noun, verb))
	if !ok {
		return nil, fmt.Errorf("Can't find scheme for noun %v, verb %v.", noun, verb)
	}
	return scheme, nil
}

func (c *C) DeleteFile(a iface.Filter, data map[string]interface{}) error {
	upd := map[string]interface{}{
		"$pull": map[string]interface{}{
			data["key"].(string): data["file"],
		},
	}
	return a.Update(upd)
}

func (c *C) DeleteAllFiles(a iface.Filter, data map[string]interface{}) error {
	upd := map[string]interface{}{
		"$unset": data["key"].(string),		// We don't care about the files themselves.
	}
	return a.Update(upd)
}

func (c *C) New(a iface.Filter) ([]map[string]interface{}, error) {
	scheme, err := c.getScheme(a.Subject(), "Insert")
	if err != nil {
		return nil, err
	}
	return convert.SchemeToFields(scheme, nil)
}

func (c *C) Edit(a iface.Filter) ([]map[string]interface{}, error) {
	doc, err := a.FindOne()
	if err != nil {
		return nil, err
	}
	scheme, err := c.getScheme(a.Subject(), "Update")
	if err != nil {
		return nil, err
	}
	return convert.SchemeToFields(scheme, doc)
}