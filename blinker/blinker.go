package blinker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
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

func getLprJob(root, path string) *LprJob {
	parts := strings.Split(path, "-")
	if len(parts) != 3 {
		return nil
	} else {
		// open the results file .json
		raw := make(map[string]interface{})
		if jsonf, err := os.Open(filepath.Join(root, path)); err == nil {
			defer jsonf.Close()

			dec := json.NewDecoder(jsonf)
			if err = dec.Decode(&raw); err == nil {
				glog.Infoln("json = ", raw)
			} else {
				return nil
			}

			country, region, id := parts[0], parts[1], strings.Replace(parts[2], ".json", "", -1)
			img := getPath(root, country, region, id)
			hasImage := false
			if _, err := os.Stat(img); err == nil {
				hasImage = true
			}

			return &LprJob{
				Country:   country,
				Region:    region,
				Id:        id,
				Path:      path,
				RawResult: raw,
				HasImage:  hasImage,
			}
		} else {
			glog.Warningln("error", err)
		}

		return nil
	}

}

func NewService(settings Settings) (Service, error) {

	impl := &serviceImpl{
		settings: settings,
	}
	return impl, nil
}

func (this *serviceImpl) ListLprJobs() (result []*LprJob, err error) {
	list, err := ioutil.ReadDir(this.settings.FsSettings.RootDir)
	if err != nil {
		return
	}

	result = make([]*LprJob, 0)
	for _, fi := range list {
		if strings.HasSuffix(fi.Name(), ".json") {
			if job := getLprJob(this.settings.FsSettings.RootDir, fi.Name()); job != nil {
				result = append(result, job)
			}
		}
	}
	return
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

func getFormat(reader *bufio.Reader) (string, error) {
	bytes, err := reader.Peek(4)
	if len(bytes) < 4 || err != nil {
		return "", ERROR_UNKNOWN_IMAGE_FORMAT
	}
	if bytes[0] == 0x89 && bytes[1] == 0x50 && bytes[2] == 0x4E && bytes[3] == 0x47 {
		return "png", nil
	}
	if bytes[0] == 0xFF && bytes[1] == 0xD8 {
		return "jpg", nil
	}
	if bytes[0] == 0x47 && bytes[1] == 0x49 && bytes[2] == 0x46 && bytes[3] == 0x38 {
		return "gif", nil
	}
	if bytes[0] == 0x42 && bytes[1] == 0x4D {
		return "bmp", nil
	}
	return "", ERROR_UNKNOWN_IMAGE_FORMAT
}

func transcodeToPng(path string) (err error) {
	// Transcode the file to png
	glog.Infoln("Transcode to PNG:", path)
	imgfile, err := os.Open(path)
	if err != nil {
		return
	}
	img, format, err := image.Decode(imgfile)
	if err != nil {
		return
	}

	glog.Infoln("Image format is", format)
	outfile, err := os.Create(path + ".png")
	if err != nil {
		return
	}

	defer outfile.Close()
	if err := png.Encode(outfile, img); err == nil {
		path = path + ".png"
	}
	return
}

func (this *serviceImpl) RunLprJob(country, region, id string, source io.ReadCloser) (stdout []byte, err error) {

	src := bufio.NewReader(source)
	// read the format
	format, err := getFormat(src)
	if err != nil {
		return
	}

	glog.Infoln("FORMAT is", format)

	base := getPath(this.settings.FsSettings.RootDir, country, region, id)
	path := base + "." + format

	glog.Infoln("ExecAlpr: saving to file", path)
	dst, err := os.Create(path)
	_, err = io.Copy(dst, src)
	if err != nil {
		return
	} else {
		dst.Close()
	}

	alpr := &LprJob{
		Country: country,
		Region:  region,
		Id:      id,
		Path:    path,
	}

	stdout, err = alpr.Execute()
	if err != nil {
		return
	}

	// copy the results
	json, err := os.Create(base + ".json")

	defer json.Close()

	glog.Infoln("ExecAlpr: saving results to", json.Name())
	json.Write(stdout)

	// rename the path to base
	err = os.Rename(path, base)

	return
}

func (this *serviceImpl) Close() {
	glog.Infoln("Service closed")
}

func (this *LprJob) Execute() (stdout []byte, err error) {
	cmd := exec.Command("alpr", "-c", this.Country, "-t", this.Region, "-j", this.Path)
	glog.Infoln("exec command:", cmd)
	stdout, err = cmd.Output()
	glog.Infoln("exec result", string(stdout), err)
	return
}
