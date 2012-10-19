// Context contains the type Uni. An instance of this type is passed to the modules when routing the control to them.
package context

import (
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/frame/lang"
	"labix.org/v2/mgo"
	"net/http"
)

// General context for the application.
type Uni struct {
	Modifiers			map[string]interface{}
	Session 			*mgo.Session
	Db      			*mgo.Database
	W       			http.ResponseWriter
	Req     			*http.Request
	secret  			string                 		// Used for things like encryption/decryption. Basically a permanent random data.
	Path       			string                 		// Path string
	opt     			string                 		// Original string representation of the option, if one needs a version which is guaranteedly untinkered.
	Opt     			map[string]interface{} 		// Freshest options from database.
	Dat     			map[string]interface{} 		// General communication channel.
	Put     			func(...interface{})   		// Just a convenience function to allow fast output to http response.
	Root    			string                 		// Absolute path of the application.
	Ev      			iface.Event
	Route				*lang.Route
	Sentence			*lang.Sentence
	FilterCreator		func(string, map[string]interface{}) iface.Filter
	NewModule			func(string) iface.Module
}

// Set only once.
func (u *Uni) SetOriginalOpt(s string) {
	if u.opt == "" {
		u.opt = s
	}
}

func (u *Uni) OriginalOpt() string {
	return u.opt
}

// Maybe we should not even return the secret, because a badly written module can make it public.
// Or, we could serve different values to different packages.
// That makes the encrypted values noncompatible across packages though.
func (u *Uni) Secret() string {
	return u.secret
}

// Set only once.
func (u *Uni) SetSecret(s string) {
	if u.secret == "" {
		u.secret = s
	}
}