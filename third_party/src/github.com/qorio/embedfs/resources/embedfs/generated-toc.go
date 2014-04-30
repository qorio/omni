// AUTO-GENERATED TOC
// DO NOT EDIT!!!
package embedfs

import (
	"net/http"
	"os"
	embedfs "github.com/qorio/embedfs/resources"
)

func init() {

	DIR.AddDir(DIR)

}

var DIR = embedfs.DirAlloc("embedfs")

func Dir(path string) http.FileSystem {
	if handle, err := DIR.Open(); err == nil {
		return handle
	}
	return nil
}

func Mount() http.FileSystem {
	return Dir(".")
}

func FileInfo() os.FileInfo {
	return DIR
}
