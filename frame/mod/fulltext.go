package mod

import "github.com/opesun/chill/modules/fulltext"

func init() {
	mods.register("fulltext", fulltext.C{})
}