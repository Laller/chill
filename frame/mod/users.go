package mod

import "github.com/opesun/chill/modules/users"

func init() {
	mods.register("users", users.C{})
}