// Package web embeds the compiled SvelteKit SPAs so the Go binary can serve
// them without any external files: the player app (web/build) and the admin
// app (web/admin, served on the admin host). The Docker build populates both
// via `npm run build` before `go build`; the placeholders committed to the
// repo keep `go build` working without a frontend build present.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var buildFS embed.FS

//go:embed all:admin
var adminFS embed.FS

// AdminFS returns the admin app's build output rooted at its top level.
func AdminFS() fs.FS {
	sub, err := fs.Sub(adminFS, "admin")
	if err != nil {
		panic(err)
	}
	return sub
}

// DistFS returns the SvelteKit build output rooted at its top level, suitable
// for apis.Static.
func DistFS() fs.FS {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		panic(err)
	}
	return sub
}
