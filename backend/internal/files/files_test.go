package files_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files/filestest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const otherStable = "00000000-0000-4000-8000-0000000009a1"
const otherUser = "00000000-0000-4000-8000-0000000009a2"

var (
	pngBytes  = filestest.PNG()
	jpegBytes = filestest.JPEG()
	pdfBytes  = filestest.PDF()
	heicBytes = append([]byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c', 0, 0, 0, 0}, make([]byte, 32)...)
	webpBytes = filestest.WebP()
)

func setup(t *testing.T) (*pgxpool.Pool, http.Handler) {
	t.Helper()
	t.Cleanup(files.SetDir(t.TempDir()))
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO stables (id, name) VALUES ($1, 'Anderer Stall')`, otherStable); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, stable_id, name, email) VALUES ($1, $2, 'Fremd', 'fremd@example.org')`, otherUser, otherStable); err != nil {
		t.Fatal(err)
	}
	return pool, httpapi.NewHandler(httpapi.Deps{Pool: pool})
}

func upload(t *testing.T, pool *pgxpool.Pool, h http.Handler, userID, field, filename, contentType string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	hdr.Set("Content-Type", contentType)
	part, _ := mw.CreatePart(hdr)
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/api/v1/files", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if userID != "" {
		authtest.Authorize(t, pool, req, userID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func get(t *testing.T, pool *pgxpool.Pool, h http.Handler, userID, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", url, nil)
	if userID != "" {
		authtest.Authorize(t, pool, req, userID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) files.Saved {
	t.Helper()
	var s files.Saved
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return s
}

// reference makes path the photo of a blanket of Luna, so members may download it.
func reference(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `INSERT INTO blankets (stable_id, horse_id, name, photo_path) VALUES ($1, $2, 'Testdecke', $3)`,
		seed.StableB, seed.HorseLuna, path)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUploadAndDownload(t *testing.T) {
	pool, h := setup(t)
	cases := []struct {
		name string
		data []byte
		ct   string
		ext  string
	}{
		{"png", pngBytes, "image/png", ".png"},
		{"jpeg", jpegBytes, "image/jpeg", ".jpg"},
		{"pdf", pdfBytes, "application/pdf", ".pdf"},
		{"webp", webpBytes, "image/webp", ".webp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := upload(t, pool, h, seed.UserJan, "file", "../../evil name.exe", c.ct, c.data)
			if rec.Code != http.StatusCreated {
				t.Fatalf("upload = %d %s", rec.Code, rec.Body)
			}
			s := decode(t, rec)
			if !strings.HasPrefix(s.Path, seed.StableB+"/") || !strings.HasSuffix(s.Path, c.ext) {
				t.Fatalf("path = %q", s.Path)
			}
			if strings.Contains(s.Path, "evil") {
				t.Fatalf("client file name leaked into %q", s.Path)
			}
			if s.URL != "/api/v1/files/"+s.Path || s.ContentType != c.ct || s.Size != int64(len(c.data)) {
				t.Fatalf("saved = %+v", s)
			}
			// Nothing points to the file yet: nobody can load it, not even the uploader.
			if rec := get(t, pool, h, seed.UserJan, s.URL); rec.Code != http.StatusNotFound {
				t.Fatalf("unreferenced download = %d", rec.Code)
			}
			reference(t, pool, s.Path)
			dl := get(t, pool, h, seed.UserMia, s.URL)
			if dl.Code != http.StatusOK || !bytes.Equal(dl.Body.Bytes(), c.data) {
				t.Fatalf("download = %d", dl.Code)
			}
			if dl.Header().Get("Content-Type") != c.ct || dl.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("headers = %v", dl.Header())
			}
		})
	}
}

func TestUploadStripsMetadata(t *testing.T) {
	pool, h := setup(t)
	payload := []byte("Exif\x00\x00GPSLatitude 51.66 N")
	exif := append([]byte{0xFF, 0xE1, 0, byte(len(payload) + 2)}, payload...)
	withExif := append(append([]byte{}, jpegBytes[:2]...), exif...)
	withExif = append(withExif, jpegBytes[2:]...)
	withExif = append(withExif, []byte("trailing motion photo")...)

	rec := upload(t, pool, h, seed.UserJan, "file", "iphone.jpg", "image/jpeg", withExif)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	s := decode(t, rec)
	if s.Size != int64(len(jpegBytes)) {
		t.Errorf("size = %d, want %d (metadata removed)", s.Size, len(jpegBytes))
	}
	reference(t, pool, s.Path)
	dl := get(t, pool, h, seed.UserMia, s.URL)
	if dl.Code != http.StatusOK || !bytes.Equal(dl.Body.Bytes(), jpegBytes) {
		t.Fatalf("download = %d, body differs from the clean image: %v", dl.Code, !bytes.Equal(dl.Body.Bytes(), jpegBytes))
	}
	// A damaged image is refused rather than stored with unknown content.
	broken := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	if rec := upload(t, pool, h, seed.UserJan, "file", "a.png", "image/png", broken); rec.Code != http.StatusBadRequest {
		t.Errorf("corrupt png = %d %s", rec.Code, rec.Body)
	}
	entries, _ := os.ReadDir(filepath.Join(files.Dir(), seed.StableB))
	if len(entries) != 1 {
		t.Errorf("files on disk = %d, want 1 (no leftovers of the refused upload)", len(entries))
	}
}

func TestUploadRejectsWrongContent(t *testing.T) {
	pool, h := setup(t)
	// Declared as image but really HTML / an executable: sniffing decides.
	for name, data := range map[string][]byte{
		"html":  []byte("<html><script>alert(1)</script></html>"),
		"text":  []byte("just text"),
		"exe":   []byte("MZ\x90\x00\x03\x00\x00\x00"),
		"gif":   []byte("GIF89a\x01\x00\x01\x00"),
		"heic":  heicBytes, // recognised, but its metadata cannot be stripped
		"empty": {},
	} {
		rec := upload(t, pool, h, seed.UserJan, "file", "a.png", "image/png", data)
		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("%s: status = %d %s", name, rec.Code, rec.Body)
		}
	}
	// Declared type not allowed at all.
	if rec := upload(t, pool, h, seed.UserJan, "file", "a.png", "text/html", pngBytes); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("declared text/html: status = %d", rec.Code)
	}
	// Declared image/png, real JPEG: accepted, stored as the sniffed type.
	rec := upload(t, pool, h, seed.UserJan, "file", "a.png", "image/png", jpegBytes)
	if rec.Code != http.StatusCreated || decode(t, rec).ContentType != "image/jpeg" {
		t.Errorf("mismatch: %d %s", rec.Code, rec.Body)
	}
}

func TestUploadSizeLimit(t *testing.T) {
	pool, h := setup(t)
	dir := files.Dir()
	big := append(append([]byte{}, pngBytes...), make([]byte, files.MaxSize)...)
	rec := upload(t, pool, h, seed.UserJan, "file", "big.png", "image/png", big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("too large = %d", rec.Code)
	}
	// Nothing (not even a temp file) stays behind.
	entries, _ := os.ReadDir(filepath.Join(dir, seed.StableB))
	if len(entries) != 0 {
		t.Fatalf("leftover files: %v", entries)
	}
	// Exactly the limit is fine.
	ok := append(append([]byte{}, pngBytes...), make([]byte, files.MaxSize-len(pngBytes))...)
	if rec := upload(t, pool, h, seed.UserJan, "file", "ok.png", "image/png", ok); rec.Code != http.StatusCreated {
		t.Fatalf("at limit = %d %s", rec.Code, rec.Body)
	}
}

func TestUploadBadRequests(t *testing.T) {
	pool, h := setup(t)
	if rec := upload(t, pool, h, "", "file", "a.png", "image/png", pngBytes); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous upload = %d", rec.Code)
	}
	if rec := upload(t, pool, h, seed.UserJan, "other", "a.png", "image/png", pngBytes); rec.Code != http.StatusBadRequest {
		t.Errorf("wrong field = %d", rec.Code)
	}
	req := httptest.NewRequest("POST", "/api/v1/files", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	authtest.Authorize(t, pool, req, seed.UserJan)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("json body = %d", rec.Code)
	}
}

func TestDownloadAccessControl(t *testing.T) {
	pool, h := setup(t)
	s := decode(t, upload(t, pool, h, seed.UserJan, "file", "a.pdf", "application/pdf", pdfBytes))

	if rec := get(t, pool, h, "", s.URL); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous = %d", rec.Code)
	}
	// A user of another stable gets 404, not 403 (do not reveal the file).
	if rec := get(t, pool, h, otherUser, s.URL); rec.Code != http.StatusNotFound {
		t.Errorf("other stable = %d", rec.Code)
	}
	// The other stable cannot upload into ours either: the path always uses the caller's stable.
	other := decode(t, upload(t, pool, h, otherUser, "file", "a.pdf", "application/pdf", pdfBytes))
	if !strings.HasPrefix(other.Path, otherStable+"/") {
		t.Errorf("other stable path = %q", other.Path)
	}
	if rec := get(t, pool, h, seed.UserJan, other.URL); rec.Code != http.StatusNotFound {
		t.Errorf("Jan reading other stable's file = %d", rec.Code)
	}
	// Missing file in own stable.
	if rec := get(t, pool, h, seed.UserJan, "/api/v1/files/"+seed.StableB+"/"+strings.Repeat("a", 32)+".png"); rec.Code != http.StatusNotFound {
		t.Errorf("missing = %d", rec.Code)
	}
}

func TestDownloadPathTraversal(t *testing.T) {
	pool, h := setup(t)
	// A secret next to the upload dir and one in another stable's folder.
	secret := filepath.Join(filepath.Dir(files.Dir()), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(secret)
	bad := []string{
		"../secret.txt",
		seed.StableB + "/../../secret.txt",
		seed.StableB + "/..%2f..%2fsecret.txt",
		"%2e%2e/secret.txt",
		seed.StableB + "/" + strings.Repeat("a", 32) + ".png/../../x",
		seed.StableB + "/.upload-123",
		seed.StableB + "/" + strings.Repeat("a", 32) + ".exe",
		seed.StableB,
		strings.ToUpper(otherStable) + "/" + strings.Repeat("a", 32) + ".png",
		"\\..\\secret.txt",
	}
	for _, p := range bad {
		rec := get(t, pool, h, seed.UserJan, "/api/v1/files/"+p)
		if strings.Contains(rec.Body.String(), "secret") && !strings.Contains(rec.Body.String(), "error") && rec.Code == http.StatusOK {
			t.Errorf("%q leaked: %d %s", p, rec.Code, rec.Body)
		}
		// ServeMux cleans ".." paths with a redirect (307) before the handler runs; the
		// handler itself answers 404 for everything that is not stable_id/random.ext.
		switch rec.Code {
		case http.StatusNotFound, http.StatusBadRequest, http.StatusTemporaryRedirect, http.StatusMovedPermanently:
		default:
			t.Errorf("%q = %d", p, rec.Code)
		}
	}
	for _, p := range []string{"../secret.txt", "../../etc/passwd", "a/b/c"} {
		if files.Belongs(seed.StableB, p) {
			t.Errorf("Belongs(%q) = true", p)
		}
		if _, _, err := files.Open(seed.StableB, p); err == nil {
			t.Errorf("Open(%q) succeeded", p)
		}
		if err := files.Remove(seed.StableB, p); err == nil {
			t.Errorf("Remove(%q) succeeded", p)
		}
	}
}

func TestDownloadFollowsReferences(t *testing.T) {
	pool, h := setup(t)
	ctx := context.Background()
	doc := decode(t, upload(t, pool, h, seed.UserJan, "file", "pass.pdf", "application/pdf", pdfBytes))
	obs := decode(t, upload(t, pool, h, seed.UserTom, "file", "a.png", "image/png", pngBytes))
	loose := decode(t, upload(t, pool, h, seed.UserJan, "file", "b.png", "image/png", pngBytes))
	for _, q := range []string{
		`INSERT INTO horse_documents (stable_id, horse_id, kind, title, file_path, uploaded_by) VALUES ('` + seed.StableB + `', '` + seed.HorseLuna + `', 'passport', 'Pass', '` + doc.Path + `', '` + seed.UserJan + `')`,
		`INSERT INTO observations (stable_id, horse_id, reported_by, media) VALUES ('` + seed.StableB + `', '` + seed.HorseLuna + `', '` + seed.UserTom + `', '["` + obs.Path + `"]')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// A document follows the document rule: owner, rider and admin of the horse; a plain
	// member and the owner of another horse see a 404.
	for user, want := range map[string]int{seed.UserJan: 200, seed.UserMia: 200, seed.UserTom: 404, seed.UserAnna: 404, otherUser: 404} {
		if rec := get(t, pool, h, user, doc.URL); rec.Code != want {
			t.Errorf("document as %s = %d, want %d", user, rec.Code, want)
		}
	}
	// An observation photo is visible to the whole stable.
	for user, want := range map[string]int{seed.UserTom: 200, seed.UserAnna: 200, otherUser: 404} {
		if rec := get(t, pool, h, user, obs.URL); rec.Code != want {
			t.Errorf("observation photo as %s = %d, want %d", user, rec.Code, want)
		}
	}
	// A file nothing points to is served to nobody.
	if rec := get(t, pool, h, seed.UserJan, loose.URL); rec.Code != http.StatusNotFound {
		t.Errorf("loose file = %d", rec.Code)
	}
	// Detaching the document takes the file away again.
	if _, err := pool.Exec(ctx, `DELETE FROM horse_documents WHERE file_path = $1`, doc.Path); err != nil {
		t.Fatal(err)
	}
	if rec := get(t, pool, h, seed.UserJan, doc.URL); rec.Code != http.StatusNotFound {
		t.Errorf("detached document = %d", rec.Code)
	}
}

type link struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

func postLink(t *testing.T, pool *pgxpool.Pool, h http.Handler, userID, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/files/download-link", strings.NewReader(`{"url":"`+url+`"}`))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		authtest.Authorize(t, pool, req, userID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDownloadLink(t *testing.T) {
	pool, h := setup(t)
	s := decode(t, upload(t, pool, h, seed.UserJan, "file", "a.png", "image/png", pngBytes))
	reference(t, pool, s.Path)

	rec := postLink(t, pool, h, seed.UserMia, s.URL)
	if rec.Code != http.StatusOK {
		t.Fatalf("download-link = %d %s", rec.Code, rec.Body)
	}
	var l link
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(l.URL, s.URL+"?dl=") || l.ExpiresAt == "" {
		t.Fatalf("link = %+v", l)
	}
	// The link works without a session and only for its file.
	if rec := get(t, pool, h, "", l.URL); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), pngBytes) {
		t.Errorf("link download = %d", rec.Code)
	}
	token := strings.TrimPrefix(l.URL, s.URL+"?dl=")
	otherFile := decode(t, upload(t, pool, h, seed.UserJan, "file", "b.png", "image/png", pngBytes))
	reference(t, pool, otherFile.Path)
	if rec := get(t, pool, h, "", otherFile.URL+"?dl="+token); rec.Code != http.StatusUnauthorized {
		t.Errorf("link on another file = %d", rec.Code)
	}
	if rec := get(t, pool, h, "", s.URL+"?dl="+token[:len(token)-2]+"xx"); rec.Code != http.StatusUnauthorized {
		t.Errorf("tampered link = %d", rec.Code)
	}
	if rec := get(t, pool, h, "", s.URL+"?dl=nope"); rec.Code != http.StatusUnauthorized {
		t.Errorf("garbage link = %d", rec.Code)
	}
	// The link carries the user: another stable's member gets a link that finds nothing.
	rec = postLink(t, pool, h, otherUser, s.URL)
	if rec.Code != http.StatusOK {
		t.Fatalf("download-link other = %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &l)
	if rec := get(t, pool, h, "", l.URL); rec.Code != http.StatusNotFound {
		t.Errorf("other stable's link = %d", rec.Code)
	}
	// Session tokens are never accepted in the URL.
	if rec := get(t, pool, h, "", s.URL+"?access_token="+authtest.Token(t, pool, seed.UserMia)); rec.Code != http.StatusUnauthorized {
		t.Errorf("access_token = %d", rec.Code)
	}
	// Only file routes can be linked, and only signed-in members ask for links.
	for _, bad := range []string{"/api/v1/horses", "https://evil.example/" + s.URL, s.URL + "?x=1", "/api/v1/files/../secret"} {
		if rec := postLink(t, pool, h, seed.UserMia, bad); rec.Code != http.StatusBadRequest {
			t.Errorf("link for %q = %d, want 400", bad, rec.Code)
		}
	}
	if rec := postLink(t, pool, h, "", s.URL); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous download-link = %d", rec.Code)
	}
	req := httptest.NewRequest("GET", "/api/v1/horses?dl="+token, nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("dl on another route = %d", rec.Code)
	}
}

func TestSaveGoAPI(t *testing.T) {
	t.Cleanup(files.SetDir(t.TempDir()))
	ctx := context.Background()
	s, err := files.Save(ctx, seed.StableB, bytes.NewReader(pngBytes), "")
	if err != nil {
		t.Fatal(err)
	}
	if !files.Belongs(seed.StableB, s.Path) || files.Belongs(otherStable, s.Path) {
		t.Fatalf("Belongs wrong for %q", s.Path)
	}
	f, ct, err := files.Open(seed.StableB, s.Path)
	if err != nil || ct != "image/png" {
		t.Fatalf("Open = %v %q", err, ct)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if !bytes.Equal(got, pngBytes) {
		t.Fatal("content differs")
	}
	if _, _, err := files.Open(otherStable, s.Path); err != files.ErrInvalidPath {
		t.Fatalf("cross stable Open = %v", err)
	}
	if err := files.Remove(seed.StableB, s.Path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := files.Open(seed.StableB, s.Path); err != files.ErrNotFound {
		t.Fatalf("after Remove = %v", err)
	}
	if _, err := files.Save(ctx, "not-a-uuid", bytes.NewReader(pngBytes), ""); err == nil {
		t.Fatal("invalid stable id accepted")
	}
	if _, err := files.Save(ctx, seed.StableB, bytes.NewReader([]byte("hello")), "image/png"); err != files.ErrUnsupportedType {
		t.Fatalf("text = %v", err)
	}
}
