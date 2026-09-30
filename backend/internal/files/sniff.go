package files

import (
	"bytes"
	"net/http"
	"strings"
)

// heicBrands are the ISO-BMFF major brands of HEIC/HEIF images (iPhone photos).
var heicBrands = [][]byte{
	[]byte("heic"), []byte("heix"), []byte("heim"), []byte("heis"),
	[]byte("hevc"), []byte("hevx"), []byte("mif1"), []byte("msf1"),
}

// Sniff returns the content type of the first bytes of a file, looking at the
// content only. It recognizes what net/http knows (JPEG, PNG, WebP, PDF) plus
// HEIC. Anything else is returned as reported by net/http and is never allowed.
func Sniff(head []byte) string {
	if len(head) >= 12 && bytes.Equal(head[4:8], []byte("ftyp")) {
		for _, b := range heicBrands {
			if bytes.Equal(head[8:12], b) {
				return "image/heic"
			}
		}
	}
	ct := http.DetectContentType(head)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return ct
}
