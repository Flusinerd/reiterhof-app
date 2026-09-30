package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// strip writes data to a file, runs the metadata filter and returns the result.
func strip(t *testing.T, data []byte, contentType string) ([]byte, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := stripMetadataFile(p, contentType); err != nil {
		return nil, err
	}
	out, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".clean"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("temporary file left behind: %v", err)
	}
	return out, nil
}

func testImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := range 3 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{uint8(x * 60), uint8(y * 80), 100, 255})
		}
	}
	return img
}

// jpegSegment builds a length-bearing segment.
func jpegSegment(marker byte, payload []byte) []byte {
	seg := []byte{0xFF, marker, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func TestStripJPEGKeepsPixelsDropsMetadata(t *testing.T) {
	var enc bytes.Buffer
	if err := jpeg.Encode(&enc, testImage(), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	original := enc.Bytes()
	if !bytes.HasPrefix(original, []byte{0xFF, 0xD8}) {
		t.Fatal("encoder output is not a JPEG")
	}
	var in bytes.Buffer
	in.Write(original[:2])
	in.Write(jpegSegment(0xE0, []byte("JFIF\x00\x01\x02\x00\x00\x01\x00\x01\x00\x00")))
	in.Write(jpegSegment(0xE1, append([]byte("Exif\x00\x00"), []byte("GPSLatitude 51.66")...)))
	in.Write(jpegSegment(0xE1, []byte("http://ns.adobe.com/xap/1.0/\x00<x:xmpmeta>location</x:xmpmeta>")))
	in.Write(jpegSegment(0xE2, append([]byte("ICC_PROFILE\x00"), make([]byte, 20)...)))
	in.Write(jpegSegment(0xE2, []byte("MPF\x00second image")))
	in.Write(jpegSegment(0xED, []byte("Photoshop 3.0\x00IPTC caption")))
	in.Write(jpegSegment(0xEE, []byte("Adobe\x00\x64\x00\x00\x00\x00\x01")))
	in.Write(jpegSegment(0xFE, []byte("a comment naming the photographer")))
	in.Write(original[2:])
	in.Write([]byte("...ftypmp4 motion photo trailer with audio"))

	out, err := strip(t, in.Bytes(), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"Exif", "GPSLatitude", "xmpmeta", "MPF", "IPTC", "photographer", "motion photo"} {
		if bytes.Contains(out, []byte(gone)) {
			t.Errorf("%q survived", gone)
		}
	}
	for _, kept := range []string{"JFIF\x00", "ICC_PROFILE\x00", "Adobe"} {
		if !bytes.Contains(out, []byte(kept)) {
			t.Errorf("%q was dropped", kept)
		}
	}
	if !bytes.HasSuffix(out, []byte{0xFF, 0xD9}) {
		t.Error("output does not end with EOI")
	}
	want, err := jpeg.Decode(bytes.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}
	got, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("stripped JPEG does not decode: %v", err)
	}
	if got.Bounds() != want.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
	}
	for y := range 3 {
		for x := range 4 {
			if got.At(x, y) != want.At(x, y) {
				t.Fatalf("pixel %d,%d differs", x, y)
			}
		}
	}
}

func TestStripJPEGProgressiveStructure(t *testing.T) {
	// A synthetic multi-scan file: the filter only follows the marker structure. Entropy
	// data with stuffed 0xFF00 and restart markers must pass unchanged, a DHT between two
	// scans and the second scan must stay, the comment must go.
	entropy1 := []byte{0x12, 0xFF, 0x00, 0x34, 0xFF, 0xD0, 0x56}
	entropy2 := []byte{0x78, 0xFF, 0x00, 0x9A}
	var in, want bytes.Buffer
	for _, b := range []*bytes.Buffer{&in, &want} {
		b.Write([]byte{0xFF, 0xD8})
		b.Write(jpegSegment(0xDB, []byte{0, 1, 2}))
		b.Write(jpegSegment(0xC2, []byte{8, 0, 3, 0, 4, 1, 1, 0x11, 0}))
	}
	in.Write(jpegSegment(0xFE, []byte("comment")))
	for _, b := range []*bytes.Buffer{&in, &want} {
		b.Write(jpegSegment(0xDA, []byte{1, 1, 0, 0, 0, 0}))
		b.Write(entropy1)
		b.Write(jpegSegment(0xC4, []byte{0x10, 1, 2}))
		b.Write(jpegSegment(0xDA, []byte{1, 1, 0, 1, 5, 0}))
		b.Write(entropy2)
		b.Write([]byte{0xFF, 0xD9})
	}
	in.Write([]byte("trailing"))
	out, err := strip(t, in.Bytes(), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, want.Bytes()) {
		t.Errorf("output\n%x\nwant\n%x", out, want.Bytes())
	}
}

func TestStripJPEGRejectsCorruptFiles(t *testing.T) {
	for name, data := range map[string][]byte{
		"not a jpeg":           []byte("hello world"),
		"truncated segment":    {0xFF, 0xD8, 0xFF, 0xE1, 0x00},
		"segment longer than":  append([]byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x10}, 1, 2),
		"reserved marker":      {0xFF, 0xD8, 0xFF, 0x05, 0x00, 0x02},
		"ends before any scan": append([]byte{0xFF, 0xD8}, jpegSegment(0xDB, []byte{1})...),
		"zero length":          {0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x01},
		"garbage after marker": {0xFF, 0xD8, 0xFF, 0x00},
	} {
		if _, err := strip(t, data, "image/jpeg"); !errors.Is(err, errCorrupt) {
			t.Errorf("%s: err = %v, want errCorrupt", name, err)
		}
	}
	// A scan cut off without EOI is a truncated photo, not an attack: the prefix is kept.
	in := append([]byte{0xFF, 0xD8}, jpegSegment(0xDA, []byte{1})...)
	in = append(in, 1, 2, 3)
	if out, err := strip(t, in, "image/jpeg"); err != nil || !bytes.Equal(out, in) {
		t.Errorf("truncated scan: %v %x", err, out)
	}
}

func pngChunk(typ string, data []byte) []byte {
	b := make([]byte, 8+len(data)+4)
	binary.BigEndian.PutUint32(b, uint32(len(data)))
	copy(b[4:], typ)
	copy(b[8:], data)
	crc := crc32.NewIEEE()
	crc.Write(b[4 : 8+len(data)])
	binary.BigEndian.PutUint32(b[8+len(data):], crc.Sum32())
	return b
}

func TestStripPNG(t *testing.T) {
	var enc bytes.Buffer
	if err := png.Encode(&enc, testImage()); err != nil {
		t.Fatal(err)
	}
	original := enc.Bytes()
	// Insert metadata after IHDR (the first chunk: 8 signature + 25 IHDR bytes).
	cut := 8 + 25
	var in bytes.Buffer
	in.Write(original[:cut])
	in.Write(pngChunk("tEXt", []byte("Author\x00Anna")))
	in.Write(pngChunk("iTXt", []byte("XML:com.adobe.xmp\x00\x00\x00\x00\x00<x:xmpmeta/>")))
	in.Write(pngChunk("eXIf", []byte("MM\x00*GPS")))
	in.Write(pngChunk("tIME", []byte{7, 0xEA, 9, 30, 12, 0, 0}))
	in.Write(pngChunk("pHYs", []byte{0, 0, 0x0B, 0x13, 0, 0, 0x0B, 0x13, 1}))
	in.Write(pngChunk("prVt", []byte("private vendor data")))
	in.Write(original[cut:])
	in.Write([]byte("trailing garbage"))

	out, err := strip(t, in.Bytes(), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"tEXt", "Anna", "iTXt", "xmpmeta", "eXIf", "tIME", "prVt", "trailing"} {
		if bytes.Contains(out, []byte(gone)) {
			t.Errorf("%q survived", gone)
		}
	}
	if !bytes.Contains(out, []byte("pHYs")) {
		t.Error("pHYs was dropped")
	}
	if !bytes.HasSuffix(out, pngChunk("IEND", nil)) {
		t.Error("output does not end with IEND")
	}
	got, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("stripped PNG does not decode: %v", err)
	}
	want, _ := png.Decode(bytes.NewReader(original))
	for y := range 3 {
		for x := range 4 {
			if got.At(x, y) != want.At(x, y) {
				t.Fatalf("pixel %d,%d differs", x, y)
			}
		}
	}
	for name, data := range map[string][]byte{
		"not a png":   []byte("hello world"),
		"no IEND":     original[:len(original)-12],
		"short chunk": append(append([]byte{}, pngSignature...), 0, 0, 0, 9, 'I', 'H', 'D', 'R', 1),
	} {
		if _, err := strip(t, data, "image/png"); !errors.Is(err, errCorrupt) {
			t.Errorf("%s: err = %v, want errCorrupt", name, err)
		}
	}
}

func webpChunk(fourcc string, data []byte) []byte {
	b := make([]byte, 8, 8+len(data)+1)
	copy(b, fourcc)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(data)))
	b = append(b, data...)
	if len(data)%2 == 1 {
		b = append(b, 0)
	}
	return b
}

func webpFile(chunks ...[]byte) []byte {
	var body bytes.Buffer
	body.WriteString("WEBP")
	for _, c := range chunks {
		body.Write(c)
	}
	head := make([]byte, 8)
	copy(head, "RIFF")
	binary.LittleEndian.PutUint32(head[4:], uint32(body.Len()))
	return append(head, body.Bytes()...)
}

func TestStripWebP(t *testing.T) {
	vp8x := []byte{0x0C | 0x20, 0, 0, 0, 3, 0, 0, 2, 0, 0} // EXIF, XMP and ICC flags set
	icc := webpChunk("ICCP", []byte("profile"))
	exif := webpChunk("EXIF", []byte("MM\x00*GPSLatitude")) // odd length: padded
	img := webpChunk("VP8 ", []byte{0x10, 0x02, 0x00, 0x9d, 0x01, 0x2a, 4, 0, 3, 0, 1, 2})
	xmp := webpChunk("XMP ", []byte("<x:xmpmeta/>"))
	in := append(webpFile(webpChunk("VP8X", vp8x), icc, exif, img, xmp), []byte("trailing")...)

	out, err := strip(t, in, "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	cleared := append([]byte{}, vp8x...)
	cleared[0] = 0x20
	want := webpFile(webpChunk("VP8X", cleared), icc, img)
	if !bytes.Equal(out, want) {
		t.Errorf("output\n%x\nwant\n%x", out, want)
	}
	for name, data := range map[string][]byte{
		"not webp":         []byte("RIFF\x04\x00\x00\x00WAVE"),
		"chunk too long":   webpFile(append([]byte("VP8 \xff\x00\x00\x00"), 1, 2)),
		"riff size beyond": append([]byte("RIFF\x40\x00\x00\x00WEBP"), webpChunk("VP8 ", []byte{1, 2})...),
	} {
		if _, err := strip(t, data, "image/webp"); !errors.Is(err, errCorrupt) {
			t.Errorf("%s: err = %v, want errCorrupt", name, err)
		}
	}
}

func TestStripLeavesOtherTypesAlone(t *testing.T) {
	pdf := []byte("%PDF-1.7\n/Author (Anna)\n")
	out, err := strip(t, pdf, "application/pdf")
	if err != nil || !bytes.Equal(out, pdf) {
		t.Errorf("pdf changed: %v %q", err, out)
	}
}

// The filters must not depend on reader buffering: a segment larger than the bufio buffer.
func TestStripJPEGLargeSegment(t *testing.T) {
	big := append([]byte("ICC_PROFILE\x00"), make([]byte, 40000)...)
	var in bytes.Buffer
	in.Write([]byte{0xFF, 0xD8})
	in.Write(jpegSegment(0xE2, big))
	in.Write(jpegSegment(0xE1, append([]byte("Exif\x00\x00"), make([]byte, 30000)...)))
	in.Write(jpegSegment(0xDA, []byte{1}))
	in.Write(bytes.Repeat([]byte{0x11}, 5000))
	in.Write([]byte{0xFF, 0xD9})
	out, err := strip(t, in.Bytes(), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2+4+len(big)+4+1+5000+2 {
		t.Errorf("len = %d", len(out))
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Error("Exif survived")
	}
}
