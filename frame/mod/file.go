package mod

import "github.com/opesun/chill/modules/file"

func init() {
	mods.register("file", file.C{})
}