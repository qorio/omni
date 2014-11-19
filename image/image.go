package image

import (
	"bufio"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
)

var (
	ErrUnknownFormat = errors.New("unknown-image-format")
)

func GetFormat(reader *bufio.Reader) (string, error) {
	bytes, err := reader.Peek(4)
	if len(bytes) < 4 || err != nil {
		return "", ErrUnknownFormat
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
	return "", ErrUnknownFormat
}

func TranscodeToPng(path string) (err error) {
	imgfile, err := os.Open(path)
	if err != nil {
		return
	}
	img, _, err := image.Decode(imgfile)
	if err != nil {
		return
	}
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
