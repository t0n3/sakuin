package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var assets embed.FS

func AssetsHandler() http.Handler {
	fsys := fs.FS(assets)
	assetsStatic, _ := fs.Sub(fsys, "dist")
	return http.FileServer(http.FS(assetsStatic))
}
