package files

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// Metadata stripping (privacy). A photo from a phone carries EXIF (GPS position, time, camera,
// owner name), XMP and IPTC blocks, vendor data, thumbnails and sometimes whole extra files
// after the image (Motion Photos append an MP4 with audio). None of that is needed to show
// the picture, but all of it would be served to everybody who may see the photo. Everything a
// decoder does not need is therefore removed before the file is stored; formats that cannot
// be rewritten this way (HEIC) are not accepted at all, PDFs are stored as they are.
//
// The filters copy the container structure and never touch the compressed image data, so the
// pixels are exactly what was uploaded.

// errCorrupt means the container structure could not be parsed; such a file is rejected
// rather than stored with unknown content (Save maps it to ErrCorrupt).
var errCorrupt = errors.New("files: corrupt image container")

// stripMetadataFile rewrites the image at path without metadata (JPEG, PNG and WebP; other
// types are left untouched). It writes a sibling file and renames it over the original.
func stripMetadataFile(path, contentType string) error {
	var strip func(*os.File, *bufio.Writer) error
	switch contentType {
	case "image/jpeg":
		strip = stripJPEG
	case "image/png":
		strip = stripPNG
	case "image/webp":
		strip = stripWebP
	default:
		return nil
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	cleanName := path + ".clean"
	dst, err := os.OpenFile(cleanName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(dst)
	err = strip(src, w)
	if err == nil {
		err = w.Flush()
	}
	if err != nil {
		_ = dst.Close()
		_ = os.Remove(cleanName)
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(cleanName)
		return err
	}
	if err := os.Rename(cleanName, path); err != nil {
		_ = os.Remove(cleanName)
		return err
	}
	return nil
}

// --- JPEG ---------------------------------------------------------------------------------

const (
	jpegTEM   = 0x01
	jpegSOI   = 0xD8
	jpegEOI   = 0xD9
	jpegSOS   = 0xDA
	jpegAPP0  = 0xE0
	jpegAPP2  = 0xE2
	jpegAPP14 = 0xEE
	jpegCOM   = 0xFE
)

// jpegKeepSegment decides for a length-bearing segment whether it is needed to decode the
// image. Only the JFIF header, the ICC colour profile and Adobe's colour transform flag
// survive among the application segments; Exif and XMP (APP1), IPTC/Photoshop (APP13),
// vendor blocks (other APPn), MPF multi-picture blocks (APP2) and comments are dropped.
func jpegKeepSegment(marker byte, payload []byte) bool {
	switch marker {
	case jpegAPP0:
		return bytes.HasPrefix(payload, []byte("JFIF\x00"))
	case jpegAPP2:
		return bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00"))
	case jpegAPP14:
		return bytes.HasPrefix(payload, []byte("Adobe"))
	case jpegCOM:
		return false
	}
	if marker >= 0xE0 && marker <= 0xEF { // other APPn
		return false
	}
	return true
}

// jpegReadMarker reads the next marker (0xFF plus one byte), skipping fill bytes.
func jpegReadMarker(r *bufio.Reader) (byte, error) {
	b, err := r.ReadByte()
	if err != nil || b != 0xFF {
		return 0, errCorrupt
	}
	for b == 0xFF {
		if b, err = r.ReadByte(); err != nil {
			return 0, errCorrupt
		}
	}
	if b == 0x00 {
		return 0, errCorrupt
	}
	return b, nil
}

// jpegCopyEntropy copies entropy-coded data up to the next marker and returns that marker
// (not written). Stuffed 0xFF00 and restart markers are part of the data. io.EOF means the
// stream ended without a marker (a truncated file: the copied prefix is kept).
func jpegCopyEntropy(r *bufio.Reader, w *bufio.Writer) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, io.EOF
		}
		if b != 0xFF {
			if err := w.WriteByte(b); err != nil {
				return 0, err
			}
			continue
		}
		next, err := r.ReadByte()
		if err != nil {
			return 0, io.EOF
		}
		for next == 0xFF { // fill bytes before a marker
			if next, err = r.ReadByte(); err != nil {
				return 0, io.EOF
			}
		}
		if next == 0x00 || (next >= 0xD0 && next <= 0xD7) {
			if _, err := w.Write([]byte{0xFF, next}); err != nil {
				return 0, err
			}
			continue
		}
		return next, nil
	}
}

func stripJPEG(src *os.File, w *bufio.Writer) error {
	r := bufio.NewReader(src)
	var soi [2]byte
	if _, err := io.ReadFull(r, soi[:]); err != nil || soi != [2]byte{0xFF, jpegSOI} {
		return errCorrupt
	}
	if _, err := w.Write(soi[:]); err != nil {
		return err
	}
	var (
		marker byte
		err    error
		have   bool // marker already read by jpegCopyEntropy
	)
	for {
		if !have {
			if marker, err = jpegReadMarker(r); err != nil {
				return err
			}
		}
		have = false
		switch {
		case marker == jpegEOI:
			// Everything after the end of image (trailers, embedded videos) is dropped.
			_, err := w.Write([]byte{0xFF, jpegEOI})
			return err
		case marker == jpegSOI || marker == jpegTEM || (marker >= 0xD0 && marker <= 0xD7):
			if _, err := w.Write([]byte{0xFF, marker}); err != nil {
				return err
			}
		case marker < 0xC0:
			return errCorrupt // reserved markers never occur in a valid file
		default:
			var lenBytes [2]byte
			if _, err := io.ReadFull(r, lenBytes[:]); err != nil {
				return errCorrupt
			}
			length := int(binary.BigEndian.Uint16(lenBytes[:]))
			if length < 2 {
				return errCorrupt
			}
			payloadLen := length - 2
			if marker == jpegSOS {
				if _, err := w.Write([]byte{0xFF, marker, lenBytes[0], lenBytes[1]}); err != nil {
					return err
				}
				if _, err := io.CopyN(w, r, int64(payloadLen)); err != nil {
					return errCorrupt
				}
				next, err := jpegCopyEntropy(r, w)
				if errors.Is(err, io.EOF) {
					return nil
				}
				if err != nil {
					return err
				}
				marker, have = next, true
				continue
			}
			head, _ := r.Peek(min(payloadLen, 16))
			if jpegKeepSegment(marker, head) {
				if _, err := w.Write([]byte{0xFF, marker, lenBytes[0], lenBytes[1]}); err != nil {
					return err
				}
				if _, err := io.CopyN(w, r, int64(payloadLen)); err != nil {
					return errCorrupt
				}
			} else if _, err := io.CopyN(io.Discard, r, int64(payloadLen)); err != nil {
				return errCorrupt
			}
		}
	}
}

// --- PNG ----------------------------------------------------------------------------------

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// pngKeepChunks are the ancillary chunks a decoder uses for colour, transparency, size and
// animation. Text (tEXt, zTXt, iTXt, which also carries XMP), eXIf, tIME and every other or
// private ancillary chunk is dropped. Critical chunks (upper-case first letter) always stay.
var pngKeepChunks = map[string]bool{
	"tRNS": true, "gAMA": true, "cHRM": true, "sRGB": true, "iCCP": true, "sBIT": true,
	"bKGD": true, "hIST": true, "pHYs": true, "sPLT": true,
	"acTL": true, "fcTL": true, "fdAT": true, // APNG
	"cICP": true, "mDCv": true, "cLLi": true, // HDR colour description
}

func stripPNG(src *os.File, w *bufio.Writer) error {
	r := bufio.NewReader(src)
	sig := make([]byte, len(pngSignature))
	if _, err := io.ReadFull(r, sig); err != nil || !bytes.Equal(sig, pngSignature) {
		return errCorrupt
	}
	if _, err := w.Write(sig); err != nil {
		return err
	}
	for {
		var head [8]byte
		if _, err := io.ReadFull(r, head[:]); err != nil {
			return errCorrupt // ended before IEND
		}
		length := binary.BigEndian.Uint32(head[:4])
		if length > 1<<31-1 {
			return errCorrupt
		}
		typ := string(head[4:8])
		critical := typ[0] >= 'A' && typ[0] <= 'Z'
		body := int64(length) + 4 // data plus CRC
		if critical || pngKeepChunks[typ] {
			if _, err := w.Write(head[:]); err != nil {
				return err
			}
			if _, err := io.CopyN(w, r, body); err != nil {
				return errCorrupt
			}
		} else if _, err := io.CopyN(io.Discard, r, body); err != nil {
			return errCorrupt
		}
		if typ == "IEND" {
			return nil // trailing data is dropped
		}
	}
}

// --- WebP ---------------------------------------------------------------------------------

// webpKeepChunks are the chunks of the image itself; EXIF, "XMP " and unknown chunks go.
var webpKeepChunks = map[string]bool{
	"VP8 ": true, "VP8L": true, "VP8X": true, "ALPH": true, "ANIM": true, "ANMF": true, "ICCP": true,
}

// webpMetadataFlags are the VP8X flag bits announcing EXIF (0x08) and XMP (0x04) chunks.
const webpMetadataFlags = 0x08 | 0x04

// stripWebP walks the RIFF chunks twice: the first pass measures the result, because the
// size in the RIFF header must describe it; the second pass writes it.
func stripWebP(src *os.File, w *bufio.Writer) error {
	size, err := webpChunks(bufio.NewReader(src), nil)
	if err != nil {
		return err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var head [12]byte
	copy(head[:], "RIFF")
	binary.LittleEndian.PutUint32(head[4:8], uint32(4+size))
	copy(head[8:], "WEBP")
	if _, err := w.Write(head[:]); err != nil {
		return err
	}
	_, err = webpChunks(bufio.NewReader(src), w)
	return err
}

// webpChunks reads the container and hands the kept chunks to w (nil: only count). It
// returns the number of bytes the kept chunks take. Data after the RIFF chunk is dropped.
func webpChunks(r *bufio.Reader, w *bufio.Writer) (int64, error) {
	var head [12]byte
	if _, err := io.ReadFull(r, head[:]); err != nil || string(head[:4]) != "RIFF" || string(head[8:]) != "WEBP" {
		return 0, errCorrupt
	}
	riffSize := int64(binary.LittleEndian.Uint32(head[4:8]))
	if riffSize < 4 {
		return 0, errCorrupt
	}
	var written int64
	remaining := riffSize - 4 // after "WEBP"
	for remaining > 0 {
		if remaining < 8 {
			return 0, errCorrupt
		}
		var ch [8]byte
		if _, err := io.ReadFull(r, ch[:]); err != nil {
			return 0, errCorrupt
		}
		fourcc := string(ch[:4])
		size := int64(binary.LittleEndian.Uint32(ch[4:8]))
		padded := size + size%2
		remaining -= 8
		if padded > remaining {
			return 0, errCorrupt
		}
		remaining -= padded
		if !webpKeepChunks[fourcc] {
			if _, err := io.CopyN(io.Discard, r, padded); err != nil {
				return 0, errCorrupt
			}
			continue
		}
		written += 8 + padded
		if w == nil {
			if _, err := io.CopyN(io.Discard, r, padded); err != nil {
				return 0, errCorrupt
			}
			continue
		}
		if _, err := w.Write(ch[:]); err != nil {
			return 0, err
		}
		if fourcc == "VP8X" && size >= 1 {
			flags, err := r.ReadByte()
			if err != nil {
				return 0, errCorrupt
			}
			if err := w.WriteByte(flags &^ webpMetadataFlags); err != nil {
				return 0, err
			}
			padded--
		}
		if _, err := io.CopyN(w, r, padded); err != nil {
			return 0, errCorrupt
		}
	}
	return written, nil
}
