package blinker

import (
	"fmt"
	"github.com/golang/glog"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type serviceImpl struct {
	settings Settings
}

func getPath(root, country, region, id string) string {
	return filepath.Join(root, fmt.Sprintf("%s-%s-%s", country, region, id))
}

func NewService(settings Settings) (Service, error) {

	impl := &serviceImpl{
		settings: settings,
	}
	return impl, nil
}

func (this *serviceImpl) GetImage(country, region, id string) (bytes io.ReadCloser, size int64, err error) {
	path := getPath(this.settings.FsSettings.RootDir, country, region, id)
	glog.Infoln("Reading from file", path)

	f, err := os.Open(path)
	if err != nil {
		return
	}

	stat, err := f.Stat()
	if err != nil {
		return
	}

	bytes = f
	size = stat.Size()
	return
}

func (this *serviceImpl) ExecAlpr(country, region, id string, src io.ReadCloser) (stdout []byte, err error) {
	path := getPath(this.settings.FsSettings.RootDir, country, region, id)

	glog.Infoln("ExecAlpr: saving to file", path)

	dst, err := os.Create(path)
	defer dst.Close()

	if err != nil {
		return
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		return
	}

	// If we can't tell by the extension
	jsonBase := path
	if filepath.Ext(path) == "" {
		// Transcode the file to png
		glog.Infoln("Transcode to PNG:", path)
		if imgfile, err := os.Open(path); err == nil {
			if img, format, err := image.Decode(imgfile); err == nil {
				glog.Infoln("Image ", path, "is", format)
				if outfile, err := os.Create(path + ".png"); err == nil {
					defer outfile.Close()
					if err := png.Encode(outfile, img); err == nil {
						path = path + ".png"
					}
				}
			}
		}
	} else {
		jsonBase = filepath.Join(filepath.Dir(path), strings.Split(filepath.Base(path), ".")[0])
	}

	alpr := &AlprCommand{
		Country: country,
		Region:  region,
		Path:    path,
	}

	stdout, err = alpr.Execute()
	if err != nil {
		return
	}

	// copy the results
	json, err := os.Create(jsonBase + ".json")

	defer json.Close()

	glog.Infoln("ExecAlpr: saving results to", json.Name())
	json.Write(stdout)

	return
}

func (this *serviceImpl) Close() {
	glog.Infoln("Service closed")
}

func (this *AlprCommand) Execute() (stdout []byte, err error) {
	cmd := exec.Command("alpr", "-c", this.Country, "-t", this.Region, "-j", this.Path)
	glog.Infoln("exec command:", cmd)
	stdout, err = cmd.Output()
	glog.Infoln("exec result", string(stdout), err)
	return
}
