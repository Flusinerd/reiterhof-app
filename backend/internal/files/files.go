// Package files stores uploaded images and PDFs on local disk and serves them to
// the members of the stable that owns them.
//
// Layout: <upload dir>/<stable_id>/<32 hex random>.<ext>. The relative "path"
// (stable_id/name.ext) is what other tables store (horse_documents.file_path,
// blankets.photo_path, observations.media). Other packages use the Go API:
//
//	saved, err := files.Save(ctx, stableID, reader, declaredContentType)
//	// store saved.Path; the app loads saved.URL ("/api/v1/files/<path>")
//	ok := files.Belongs(stableID, path) // validate a path sent by a client
//
// Uploads are sniffed (the declared type is not trusted), limited to MaxSize and
// named randomly, so client file names never reach the disk. Images are stored without
// their metadata (EXIF position, XMP, comments, trailers; see metadata.go).
//
// Serving follows the record that references a file (least privilege, see handlers.go):
// the raw route only answers for paths a horse document, an observation or a blanket
// points to, with the visibility of that record.
package files

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// MaxSize is the largest accepted upload (20 MB).
const MaxSize = 20 << 20

// DirEnv names the environment variable with the upload directory.
const DirEnv = "REITERHOF_UPLOAD_DIR"

// DefaultDir is used when DirEnv is unset (development). Production sets
// /var/lib/reiterhof/uploads (deploy/api.env.example).
const DefaultDir = "./uploads"

var (
	// ErrTooLarge means the upload exceeds MaxSize.
	ErrTooLarge = errors.New("files: file too large")
	// ErrUnsupportedType means the content is not an allowed image or PDF.
	ErrUnsupportedType = errors.New("files: unsupported file type")
	// ErrCorrupt means the image container is damaged, so its metadata could not be
	// removed; such a file is not stored.
	ErrCorrupt = errors.New("files: corrupt image")
	// ErrInvalidPath means a path does not have the form stable_id/name.ext or
	// belongs to another stable.
	ErrInvalidPath = errors.New("files: invalid path")
	// ErrNotFound means the path is valid but no such file exists.
	ErrNotFound = errors.New("files: not found")
)

// Saved describes a stored file.
type Saved struct {
	// Path is relative to the upload dir: "<stable_id>/<random>.<ext>". Store this.
	Path string `json:"path"`
	// URL is the download route, "/api/v1/files/<path>".
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// types maps allowed (sniffed) content types to the file extension. Only image formats
// whose metadata can be stripped (metadata.go) are accepted; HEIC is recognised by Sniff
// but refused, the phone's picker delivers JPEG anyway.
var types = map[string]string{
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

var extTypes = func() map[string]string {
	m := make(map[string]string, len(types))
	for ct, ext := range types {
		m[ext] = ct
	}
	return m
}()

const uuidPattern = `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`

var (
	pathRe   = regexp.MustCompile(`^(` + uuidPattern + `)/([0-9a-f]{32})\.(jpg|png|webp|pdf)$`)
	stableRe = regexp.MustCompile(`^` + uuidPattern + `$`)
)

var (
	dirMu sync.RWMutex
	dir   string
)

// Dir returns the upload directory (REITERHOF_UPLOAD_DIR unless SetDir was used).
func Dir() string {
	dirMu.RLock()
	d := dir
	dirMu.RUnlock()
	if d != "" {
		return d
	}
	if v := os.Getenv(DirEnv); v != "" {
		return v
	}
	return DefaultDir
}

// SetDir overrides the upload directory, mainly for tests. It returns a function
// restoring the previous value.
func SetDir(d string) (restore func()) {
	dirMu.Lock()
	prev := dir
	dir = d
	dirMu.Unlock()
	return func() {
		dirMu.Lock()
		dir = prev
		dirMu.Unlock()
	}
}

// Belongs reports whether path is a well-formed file path of the given stable.
// It does not check that the file exists. Use it to validate client input.
func Belongs(stableID, path string) bool {
	m := pathRe.FindStringSubmatch(path)
	return m != nil && stableID != "" && m[1] == strings.ToLower(stableID)
}

// ContentTypeOf returns the content type implied by a valid path's extension.
func ContentTypeOf(path string) (string, bool) {
	m := pathRe.FindStringSubmatch(path)
	if m == nil {
		return "", false
	}
	return extTypes[m[3]], true
}

// Save stores the content of r for the stable. declaredType is the type the
// client claims (may be empty or application/octet-stream); if given it must be
// an allowed type, but the stored type is always the sniffed one. The file is
// streamed to disk; more than MaxSize bytes give ErrTooLarge and keep nothing.
// Images are rewritten without metadata; a damaged image gives ErrCorrupt.
func Save(ctx context.Context, stableID string, r io.Reader, declaredType string) (Saved, error) {
	if !stableRe.MatchString(strings.ToLower(stableID)) {
		return Saved{}, fmt.Errorf("files: invalid stable id %q", stableID)
	}
	stableID = strings.ToLower(stableID)
	if d := normalizeType(declaredType); d != "" {
		if _, ok := types[d]; !ok {
			return Saved{}, ErrUnsupportedType
		}
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Saved{}, err
	}
	head = head[:n]
	ct := Sniff(head)
	ext, ok := types[ct]
	if !ok {
		return Saved{}, ErrUnsupportedType
	}

	stableDir := filepath.Join(Dir(), stableID)
	if err := os.MkdirAll(stableDir, 0o750); err != nil {
		return Saved{}, err
	}
	tmp, err := os.CreateTemp(stableDir, ".upload-*")
	if err != nil {
		return Saved{}, err
	}
	tmpName := tmp.Name()
	fail := func(err error) (Saved, error) {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return Saved{}, err
	}

	if _, err := tmp.Write(head); err != nil {
		return fail(err)
	}
	rest, err := io.Copy(tmp, io.LimitReader(r, MaxSize+1-int64(len(head))))
	if err != nil {
		return fail(err)
	}
	size := int64(len(head)) + rest
	if size > MaxSize {
		return fail(ErrTooLarge)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(0o640); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return Saved{}, err
	}
	// Privacy: the stored image carries no EXIF, XMP, IPTC, comments or trailers.
	if err := stripMetadataFile(tmpName, ct); err != nil {
		_ = os.Remove(tmpName)
		if errors.Is(err, errCorrupt) {
			return Saved{}, ErrCorrupt
		}
		return Saved{}, err
	}
	if info, err := os.Stat(tmpName); err == nil {
		size = info.Size()
	}
	name, err := randomName()
	if err != nil {
		_ = os.Remove(tmpName)
		return Saved{}, err
	}
	if err := os.Rename(tmpName, filepath.Join(stableDir, name+"."+ext)); err != nil {
		_ = os.Remove(tmpName)
		return Saved{}, err
	}
	p := stableID + "/" + name + "." + ext
	return Saved{Path: p, URL: URLFor(p), ContentType: ct, Size: size}, nil
}

// URLFor returns the download route of a stored path.
func URLFor(path string) string { return "/api/v1/files/" + path }

// Open opens a stored file of the stable and returns it with its content type.
// ErrInvalidPath for malformed paths or paths of another stable, ErrNotFound if
// the file is gone. The caller closes the file.
func Open(stableID, path string) (*os.File, string, error) {
	if !Belongs(stableID, path) {
		return nil, "", ErrInvalidPath
	}
	f, err := os.Open(filepath.Join(Dir(), filepath.FromSlash(path)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	ct, _ := ContentTypeOf(path)
	return f, ct, nil
}

// Remove deletes a stored file of the stable. A missing file is not an error.
func Remove(stableID, path string) error {
	if !Belongs(stableID, path) {
		return ErrInvalidPath
	}
	err := os.Remove(filepath.Join(Dir(), filepath.FromSlash(path)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func randomName() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func normalizeType(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch ct {
	case "", "application/octet-stream":
		return ""
	case "image/jpg", "image/pjpeg":
		return "image/jpeg"
	}
	return ct
}
