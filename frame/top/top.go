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
	"net/http"
	"net/url"
	"fmt"
	"io"
	"labix.org/v2/mgo"
	"strconv"
	"strings"
	"runtime/debug"
)

type m map[string]interface{}

var Put func(...interface{})

const (
	unfortunate_error         = "top: An unfortunate error has happened. We are deeply sorry for the inconvenience."
	no_user_module_build_hook = "top: User module does not export BuildUser hook."
)

func (t *Top) buildUser() {
	ret_rec := func(usr map[string]interface{}) {
		t.uni.Dat["_user"] = usr
	}
	ins := t.uni.NewModule("users").Instance()
	ins.Method("Init").Call(nil, t.uni)
	ins.Method("BuildUser").Call(ret_rec, filter.NewSimple(set.New(t.uni.Db, "users")))
}

// Just printing the stack trace to http response if a panic bubbles up all the way to top.
func topErr() {
	if r := recover(); r != nil {
		fmt.Println("main:", r)
		fmt.Println(string(debug.Stack()))
		Put(unfortunate_error)
		Put(fmt.Sprint("\n", r, "\n\n"+string(debug.Stack())))
	}
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
			Put(fmt.Sprint(r))
			panic(fmt.Sprint(r))
		}
	}()
	err := t.route()
	if err != nil {
		display.DErr(t.uni, err)
	}
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
		return fmt.Errorf("No nouns.")
	}
	desc, err := glue.Identify(uni.P, nouns, uni.Req.Form)
	if err != nil {
		return err
	}
	filterCreator := func(c string, input map[string]interface{}) iface.Filter {
		return filter.New(set.New(uni.Db, c), input)
	}
	inp, err := desc.CreateInputs(filterCreator)
	if err != nil {
		return err
	}
	uni.R = desc.Route
	uni.S = desc.Sentence
	module := mod.NewModule(desc.VerbLocation)
	if !module.Exists() {
		return fmt.Errorf("Unkown module.")
	}
	ins := module.Instance()
	ins.Method("Init").Call(nil, t.uni)
	ins.Method(uni.S.Verb).Call(ret_rec, inp...)
	if uni.Req.Method == "GET" {
		uni.Dat["main_noun"] = desc.Sentence.Noun
		uni.Dat["_points"] = []string{desc.VerbLocation+"/"+desc.Sentence.Verb}
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

func New(session *mgo.Session, db *mgo.Database, w http.ResponseWriter, req *http.Request, config *config.Config) *Top {
	Put = func(a ...interface{}) {
		io.WriteString(w, fmt.Sprint(a...)+"\n")
	}
	defer topErr()
	uni := &context.Uni{
		Db:      	db,
		W:       	w,
		Req:     	req,
		Put:     	Put,
		Dat:     	make(map[string]interface{}),
		Root:    	config.AbsPath,
		P:       	req.URL.Path,
		Paths:   	strings.Split(req.URL.Path, "/"),
		NewModule:	mod.NewModule,
	}
	uni.Req.ParseForm()		// Should we handle the error return of this?
	mods := modifiers(uni.Req.Form)
	uni.Modifiers = mods
	// Not sure if not giving the db session to nonadmin installations increases security, but hey, one can never be too cautious, they dont need it anyway.
	if config.DBAdmMode {
		uni.Session = session
	}
	uni.Ev = context.NewEv(uni)
	opt, opt_str, err := queryConfig(uni.Db, req.Host, config.CacheOpt) // Tricky part about the host, see comments at main_model.
	if err != nil {
		Put(err.Error())
		return &Top{}
	}
	uni.Req.Host = scut.Host(req.Host, opt)
	uni.Opt = opt
	uni.SetOriginalOpt(opt_str)
	uni.SetSecret(config.Secret)
	return &Top{uni,config}
}