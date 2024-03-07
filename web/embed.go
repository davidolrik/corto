// Package web embeds the built admin UI so the corto binary can serve it
// without external files. Run `npm run build` in this directory to populate
// the build output before compiling.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var files embed.FS

// FS returns the built admin UI as a filesystem rooted at the build output.
func FS() fs.FS {
	build, err := fs.Sub(files, "build")
	if err != nil {
		panic(err) // notest - the build directory is embedded at compile time
	}
	return build
}
