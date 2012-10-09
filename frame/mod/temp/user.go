package mod

import "github.com/opesun/chill/modules/user"

func init() {
	mods.register("user", user.C{})
}