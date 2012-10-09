// Package user implements basic user functionality.
// - Registration, deletion, update, login, logout of users.
// - Building the user itself (if logged in), and putting it to uni.Dat["_user"].
package users

import (
	"github.com/opesun/chill/frame/context"
	iface "github.com/opesun/chill/frame/interfaces"
	"github.com/opesun/chill/modules/users/model"
	"net/http"
)

type C struct {
	uni *context.Uni
}

func (c *C) Init(uni *context.Uni) {
	c.uni = uni
}

// Recover from wrong ObjectId like panics. Unset the cookie.
func unsetCookie(w http.ResponseWriter) {
	r := recover()
	if r == nil {
		return
	}
	c := &http.Cookie{Name: "user", Value: "", MaxAge: 3600000, Path: "/"}
	http.SetCookie(w, c)
}

// If there were some random database query errors or something we go on with an empty user.
func (h *C) BuildUser(a iface.Filter) (user map[string]interface{}) {
	uni := h.uni
	defer unsetCookie(uni.W)
	user = user_model.EmptyUser()
	var user_id_str string
	c, err := uni.Req.Cookie("user")
	if err != nil {
		panic(err)
	}
	user_id_str = c.Value
	block_key := []byte(uni.Secret())
	user_id, err := user_model.DecryptId(user_id_str, block_key)
	if err != nil {
		panic(err)
	}
	user, err = user_model.BuildUser(a, uni.Ev, user_id, uni.Req.Header)
	if err != nil {
		panic(err)
	}
	return
}

func (a *C) Insert(f iface.Filter, data map[string]interface{}) error {
	return f.Insert(data)
}

func (a *C) LoginForm() error {
	return nil
}

func (a *C) Login(f iface.Filter, data map[string]interface{}) error {
	if _, id, err := user_model.FindLogin(f, data["name"].(string), data["password"].(string)); err == nil {		// Maybe there could be a check here to not log in somebody who is already logged in.
		block_key := []byte(a.uni.Secret())
		return user_model.Login(a.uni.W, id, block_key)
	} else {
		return err
	}
	return nil
}

func (a *C) Logout() error {
	c := &http.Cookie{Name: "user", Value: "", Path: "/"}
	http.SetCookie(a.uni.W, c)
	return nil
}