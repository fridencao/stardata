package web

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/NYTimes/gziphandler"
)

//go:embed all:embed
var distFS embed.FS

// Handler serves the embedded local UI assets.
// Since web-local was removed, no frontend build is embedded anymore and this
// falls back to the static placeholder page in embed/index.html.
func StaticHandler() http.Handler {
	uiAssetFS := newUIAssetFS()
	return gziphandler.GzipHandler(http.FileServer(uiAssetFS))
}

// Check if an embedded dist UI exists. If not, serve the default index.html placeholder.
func newUIAssetFS() http.FileSystem {
	_, err := distFS.ReadFile("embed/dist/index.html")
	if os.IsNotExist(err) {
		return assetFS(distFS, "embed")
	}
	return assetFS(distFS, "embed/dist")
}

// Get the subtree of the embedded files with `embed` directory as a root.
func assetFS(embeddedFS embed.FS, dir string) http.FileSystem {
	subFS, err := fs.Sub(embeddedFS, dir)
	if err != nil {
		panic(fmt.Errorf("fs embed: %w", err))
	}

	return &SPARoutingFS{FileSystem: http.FS(subFS)}
}

type SPARoutingFS struct {
	FileSystem http.FileSystem
}

func (spaFS *SPARoutingFS) Open(name string) (http.File, error) {
	file, err := spaFS.FileSystem.Open(name)
	if err == nil {
		return file, nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		file, err = spaFS.FileSystem.Open("index.html")
		return file, err
	}

	return nil, err
}
