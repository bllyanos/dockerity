// Package web embeds the built UI (web/dist) into the Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Dist returns the built UI. It holds only .gitkeep until `npm run build` has run.
func Dist() fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err) // "dist" is a valid, embedded path
	}
	return sub
}
