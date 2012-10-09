package mod

import "github.com/opesun/chill/modules/example"

func init() {
	mods.register("example", example.C{})
}