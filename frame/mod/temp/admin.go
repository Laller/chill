package mod

import ad "github.com/opesun/chill/modules/admin"

func init() {
	mods.register("admin", ad.C{})
}