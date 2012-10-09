package mod

import "github.com/opesun/chill/modules/jsonedit"

func init() {
	mods.register("jsonedit", jsonedit.C{})
}