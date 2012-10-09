package skeleton

import(
	"github.com/opesun/chill/frame/context"
	"github.com/opesun/jsonp"
	"github.com/opesun/chill/frame/misc/scut"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/composables/basics"
)

type C struct {
	basics.Basics
	uni *context.Uni
}

func (c *C) Init(uni *context.Uni) {
	c.uni = uni
}

func (c *C) getScheme(subj string) map[string]interface{} {
	scheme, ok := jsonp.GetM(c.uni.Opt, "nouns."+subj+".scheme")
	if !ok {
		scheme = map[string]interface{}{
			"info": 1,
			"name": 1,
		}
	}
	return scheme
}

func (c *C) New(a iface.Filter) ([]map[string]interface{}, error) {
	scheme := c.getScheme(a.Subject())
	return scut.RulesToFields(scheme, nil)
}

func (c *C) Edit(a iface.Filter) ([]map[string]interface{}, error) {
	doc, err := a.FindOne()
	if err != nil {
		return nil, err
	}
	scheme := c.getScheme(a.Subject())
	return scut.RulesToFields(scheme, doc)
}