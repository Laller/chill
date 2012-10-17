package top

import(
	"github.com/opesun/chill/frame/context"
	"github.com/opesun/chill/frame/config"
	"github.com/opesun/chill/frame/mod"
	"github.com/opesun/chill/frame/misc/scut"
	"github.com/opesun/chill/frame/display"
	"github.com/opesun/chill/frame/filter"
	"github.com/opesun/chill/frame/set"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/verbinfo"
	"github.com/opesun/chill/frame/glue"
	"github.com/opesun/jsonp"
	"github.com/opesun/numcon"
	"github.com/opesun/sanitize"
	"net/http"
	"net/url"
	"fmt"
	"io"
	"labix.org/v2/mgo"
	"strconv"
	"strings"
)

type m map[string]interface{}

func (t *Top) buildUser() {
	ret_rec := func(usr map[string]interface{}) {
		t.uni.Dat["_user"] = usr
	}
	ins := t.uni.NewModule("users").Instance()
	ins.Method("BuildUser").Call(ret_rec, filter.NewSimple(set.New(t.uni.Db, "users")))
}

type Top struct{
	uni 	*context.Uni
	config 	*config.Config
}

func burnResults(a map[string]interface{}, key string, b []interface{}) {
	for i, v := range b {
		if i == 0 {
			a[key] = v
		} else {
			a[key+strconv.Itoa(i)] = v
		}
	}
}

func (t *Top) Get(ret []interface{}) {
	uni := t.uni
	ran := verbinfo.NewRanalyzer(ret)
	if ran.HadError() {
		display.DErr(uni, ran.Error())
		return
	}
	burnResults(uni.Dat, "main", ran.NonErrors())
	display.D(uni)
}

func (t *Top) Post(ret []interface{}) {
	uni := t.uni
	ran := verbinfo.NewRanalyzer(ret)
	var err error
	if ran.HadError() {
		err = ran.Error()
	}
	t.actionResponse(err, uni.S.Verb)
}

func (t *Top) Route() {
	defer func(){
		if r := recover(); r != nil {
			t.uni.Put(fmt.Sprint(r))
			panic(fmt.Sprint(r))
		}
	}()
	err := t.route()
	if err != nil {
		display.DErr(t.uni, err)
		return
	}
}

var opt_def = map[string]interface{}{
	"composed_of": []interface{}{"jsonedit"},
}

func (t *Top) validate(noun, verb string, data map[string]interface{}) (map[string]interface{}, error) {
	scheme_map, ok := jsonp.GetM(t.uni.Opt, fmt.Sprintf("nouns.%v.verbs.%v.input", noun, verb))
	if !ok {
		return nil, fmt.Errorf("Can't find scheme for %v %v.", noun, verb) 
	}
	ex, err := sanitize.New(scheme_map)
	if err != nil {
		return nil, err
	}
	t.uni.Ev.Fire("SanitizerMangler", ex)
	data, err = ex.Extract(data)
	if err != nil {
		return nil, err
	}
	t.uni.Ev.Fire("SanitizedDataMangler", data)
	return data, nil
}

func filterCreator(db *mgo.Database, nouns, input map[string]interface{}, c string) iface.Filter {
	return filter.New(set.New(db, c), input)
}

func (t *Top) route() error {
	uni := t.uni
	if t.config.ServeFiles && strings.Index(uni.Paths[len(uni.Paths)-1], ".") != -1 {
		t.serveFile()
		return nil
	}
	t.buildUser()
	var ret []interface{}
	ret_rec := func(i ...interface{}) {
		ret = i
	}
	nouns, ok := uni.Opt["nouns"].(map[string]interface{})
	if !ok {
		nouns = map[string]interface{}{
			"options": opt_def,
		}
	}
	if _, ok := nouns["options"]; !ok {
		nouns["options"] = opt_def
	}
	desc, err := glue.Identify(uni.P, nouns, uni.Req.Form)
	if err != nil {
		display.D(uni)
		return nil
	}
	default_level, _ := numcon.Int(uni.Opt["default_level"])
	levi, ok := jsonp.Get(uni.Opt, fmt.Sprintf("nouns.%v.verbs.%v.level", desc.Sentence.Noun, desc.Sentence.Verb))
	if !ok {
		levi = default_level
	}
	lev, _ := numcon.Int(levi)
	if scut.Ulev(uni.Dat["_user"]) < lev {
		return fmt.Errorf("Not allowed.")
	}
	filterCreator := func(c string, input map[string]interface{}) iface.Filter {
		return filterCreator(uni.Db, nouns, input, c)
	}
	inp, data, err := desc.CreateInputs(filterCreator)
	if err != nil {
		return err
	}
	if data != nil {
		if desc.Sentence.Noun != "options" {
			data, err = t.validate(desc.Sentence.Noun, desc.Sentence.Verb, data)
			if err != nil {
				return err
			}
		}
		inp = append(inp, data)
	}
	uni.R = desc.Route
	uni.S = desc.Sentence
	module := t.uni.NewModule(desc.VerbLocation)
	if !module.Exists() {
		return fmt.Errorf("Unkown module.")
	}
	ins := module.Instance()
	ins.Method("Init").Call(nil, t.uni)
	ins.Method(uni.S.Verb).Call(ret_rec, inp...)
	if uni.Req.Method == "GET" {
		uni.Dat["main_noun"] = desc.Sentence.Noun
		uni.Dat["_points"] = []string{desc.Sentence.Noun + "/" + desc.Sentence.Verb, desc.VerbLocation + "/" + desc.Sentence.Verb}
		t.Get(ret)
	} else {
		t.Post(ret)
	}
	return nil
}

// Strips information unrelated to verb input from the Form.
func modifiers(a url.Values) map[string]interface{} {
	flags := []string{"json", "src", "nofmt", "ok", "action"}
	mods := map[string]interface{}{}
	for _, v := range flags {
		if val, has := a[v]; has {
			mods[v] = val
			delete(a, v)
		}
	}
	for i, v := range a {
		if i[0] == '-' {
			mods[i[1:]] = v
			delete(a, i)
		}
	}
	return mods
}

func New(session *mgo.Session, db *mgo.Database, w http.ResponseWriter, req *http.Request, config *config.Config) (t *Top, err error) {
	put := func(a ...interface{}) {
		io.WriteString(w, fmt.Sprint(a...)+"\n")
	}
	uni := &context.Uni{
		Db:      	db,
		W:       	w,
		Req:     	req,
		Put:     	put,
		Dat:     	make(map[string]interface{}),
		Root:    	config.AbsPath,
		P:       	req.URL.Path,
		Paths:   	strings.Split(req.URL.Path, "/"),
		NewModule:	mod.NewModule,
	}
	err = uni.Req.ParseMultipartForm(1000000)		// Should we handle the error return of this?
	if err != nil {
		return nil, err
	}
	mods := modifiers(uni.Req.Form)
	uni.Modifiers = mods
	// Not sure if not giving the db session to nonadmin installations increases security, but hey, one can never be too cautious, they dont need it anyway.
	if config.DBAdmMode {
		uni.Session = session
	}
	ev := context.NewEv(uni, mod.NewModule)
	uni.Ev = ev
	uni.NewModule = ev.NewModuleProducer()
	opt, opt_str, err := queryConfig(uni.Db, req.Host, config.CacheOpt) // Tricky part about the host, see comments at main_model.
	if err != nil {
		return nil, err
	}
	uni.Req.Host = scut.Host(req.Host, opt)
	uni.Opt = opt
	uni.SetOriginalOpt(opt_str)
	uni.SetSecret(config.Secret)
	return &Top{uni,config}, nil
}