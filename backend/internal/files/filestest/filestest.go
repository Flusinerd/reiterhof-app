// Package filestest provides small valid files for tests of packages that upload through
// internal/files. Uploads are rewritten without metadata, so a file must be a well-formed
// container; the images here contain none, which keeps them byte-identical after a round trip.
package filestest

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

func tiny() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{200, 30, 30, 255})
	img.Set(1, 1, color.RGBA{30, 30, 200, 255})
	return img
}

// PNG returns a 2 x 2 PNG.
func PNG() []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, tiny()); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// JPEG returns a 2 x 2 JPEG.
func JPEG() []byte {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, tiny(), nil); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// WebP returns a minimal RIFF container with one "VP8 " chunk (the payload is not a real
// bitstream; internal/files never decodes pixels).
func WebP() []byte {
	payload := []byte{0x10, 0x02, 0x00, 0x9d, 0x01, 0x2a, 2, 0, 2, 0, 1, 2}
	body := append([]byte("WEBPVP8 "), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(body[8:], uint32(len(payload)))
	body = append(body, payload...)
	head := make([]byte, 8)
	copy(head, "RIFF")
	binary.LittleEndian.PutUint32(head[4:], uint32(len(body)))
	return append(head, body...)
}

// PDF returns a minimal PDF.
func PDF() []byte {
	return []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")
}
