// Package scut contains a somewhat ugly but useful collection of frequently appearing patterns to allow faster prototyping.
// Methods here are mainly related to view- or conroller-like parts.
package scut

import (
	"fmt"
	"github.com/opesun/numcon"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Gives you back the type of the currently used template (either "private" or public).
func TemplateType(opt map[string]interface{}) string {
	_, priv := opt["TplIsPrivate"]
	var ttype string
	if priv {
		ttype = "private"
	} else {
		ttype = "public"
	}
	return ttype
}

// Gives you back the name of the current template in use.
func TemplateName(opt map[string]interface{}) string {
	tpl, has_tpl := opt["Template"]
	if !has_tpl {
		tpl = "default"
	}
	return tpl.(string)
}

// Decides if a given relative filepath (filep) is a possible module filepath.
// This may be deprecated in the future since it seems so restrictive.
func PossibleModPath(filep string) bool {
	sl := strings.Split(filep, "/")
	return len(sl) >= 2
}

// TODO: Implement file caching here.
// Reads the fi relative filepath from either the current template, or the fallback module tpl folder if fi has at least one slash in it.
// file_reader is optional, falls back to simple ioutil.ReadFile if not given. file_reader will be a custom file_reader with caching soon.
func GetFile(root, fi string, opt map[string]interface{}, host string, file_reader func(string) ([]byte, error)) ([]byte, error) {
	if file_reader == nil {
		file_reader = ioutil.ReadFile
	}
	p := GetTPath(opt, host)
	b, err := file_reader(filepath.Join(root, p, fi))
	if err == nil {
		return b, nil
	}
	if !PossibleModPath(fi) {
		return nil, fmt.Errorf("Not found.")
	}
	mp := GetModTPath(fi)
	return file_reader(filepath.Join(root, mp[0], mp[1]))
}

func Dirify(s string) string {
	return strings.Replace(s, ":", "-", -1)
}

// Observes opt and gives you back the path of your template eg
// "templates/public/template_name" or "templates/private/hostname/template_name"
func GetTPath(opt map[string]interface{}, host string) string {
	host = Dirify(host)
	templ := TemplateName(opt)
	ttype := TemplateType(opt)
	if ttype == "public" {
		return filepath.Join("templates", ttype, templ)
	}
	return filepath.Join("templates", ttype, host, templ)
}

// Inp:	"admin/this/that.txt"
// []string{ "modules/admin/tpl", "this/that.txt"}
func GetModTPath(filename string) []string {
	sl := []string{}
	p := strings.Split(filename, "/")
	sl = append(sl, filepath.Join("modules", p[0], "tpl"))
	sl = append(sl, strings.Join(p[1:], "/"))
	return sl
}

func NotAdmin(user interface{}) bool {
	return Ulev(user) < 300
}

func IsAdmin(user interface{}) bool {
	return Ulev(user) >= 300
}

func IsModerator(user interface{}) bool {
	return Ulev(user) >= 200
}

func IsRegistered(user interface{}) bool {
	return Ulev(user) >= 100
}

func IsGuest(user interface{}) bool {
	ulev := Ulev(user)
	return (ulev > 0 && ulev < 100)
}

func IsStranger(user interface{}) bool {
	return Ulev(user) == 0
}

func SolvedPuzzles(user interface{}) bool {
	return Ulev(user) > 1
}

// Gives back the user level.
func Ulev(useri interface{}) int {
	if useri == nil {
		return 0 // useri should never be nil BTW
	}
	user := useri.(map[string]interface{})
	ulev, has := user["level"]
	if !has {
		return 0
	}
	return numcon.IntP(ulev)
}

// Merges b into a (overwriting members in a.
func Merge(a map[string]interface{}, b map[string]interface{}) {
	for i, v := range b {
		a[i] = v
	}
}

// CanonicalHost(uni.Req.Host, uni.Opt)
// Gives you back the canonical address of the site so it can be made available from different domains.
func Host(host string, opt map[string]interface{}) string {
	alias_whitelist, has_alias_whitelist := opt["host_alias_whitelist"]
	if has_alias_whitelist {
		awm := alias_whitelist.(map[string]interface{})
		if _, allowed := awm[host]; !allowed && len(awm) > 0 { // To prevent entirely locking yourself out of the site. Still can introduce problems if misused.
			panic(fmt.Sprintf("Unapproved host alias %v.", host))
		}
	}
	canon_host, has_canon := opt["canonical_host"]
	if !has_canon {
		return host
	}
	return canon_host.(string)
}

func OnlyAdmin(dat map[string]interface{}) {
	if Ulev(dat["_user"]) < 300 {
		panic("Only an admin can do this operation.")
	}
}