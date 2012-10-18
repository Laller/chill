package skeleton

import(
	"github.com/opesun/chill/frame/context"
	"github.com/opesun/jsonp"
	"github.com/opesun/chill/frame/misc/convert"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/composables/basics"
	"fmt"
)

type C struct {
	basics.Basics
	uni *context.Uni
}

func (c *C) Init(uni *context.Uni) {
	c.uni = uni
}

func (c *C) getScheme(noun, verb string) (map[string]interface{}, error) {
	scheme, ok := jsonp.GetM(c.uni.Opt, fmt.Sprintf("nouns.%v.verbs.%v.input", noun, verb))
	if !ok {
		return nil, fmt.Errorf("Can't find scheme for noun %v, verb %v.", noun, verb)
	}
	return scheme, nil
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