package files

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Register adds the upload and download routes.
//
//	POST /api/v1/files          multipart/form-data, field "file" -> 201 Saved
//	GET  /api/v1/files/{path...} download; only for members of the file's stable
//
// Errors: 400 invalid_upload, 413 file_too_large, 415 unsupported_type,
// 404 not_found (also for paths of other stables and malformed paths).
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	mux.Handle("POST /api/v1/files", auth.RequireStable(http.HandlerFunc(h.upload)))
	mux.Handle("GET /api/v1/files/{path...}", QueryToken(deps)(auth.RequireStable(http.HandlerFunc(h.download))))
}

type handler struct{ deps httpx.Deps }

func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	// Room for the multipart framing around a MaxSize file.
	r.Body = http.MaxBytesReader(w, r.Body, MaxSize+(1<<20))
	mr, err := r.MultipartReader()
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_upload", "expected multipart/form-data with a \"file\" field")
		return
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_upload", "missing \"file\" field")
			return
		}
		if err != nil {
			writeUploadError(w, h.deps, err)
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		saved, err := Save(r.Context(), user.StableID, part, part.Header.Get("Content-Type"))
		if err != nil {
			writeUploadError(w, h.deps, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, saved)
		return
	}
}

func writeUploadError(w http.ResponseWriter, deps httpx.Deps, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.Is(err, ErrTooLarge) || errors.As(err, &tooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", "file exceeds 20 MB")
	case errors.Is(err, ErrUnsupportedType):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_type", "only JPEG, PNG, WebP, HEIC and PDF are allowed")
	default:
		if deps.Log != nil {
			deps.Log.Error("files: upload", "err", err)
		}
		httpx.WriteError(w, http.StatusBadRequest, "invalid_upload", "could not read the upload")
	}
}

func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	Serve(w, r, h.deps, user.StableID, r.PathValue("path"))
}

// Serve writes the stored file of the stable, or a 404 if the path is malformed,
// belongs to another stable or does not exist. Callers must have authorized the
// user already. The content type comes from the extension chosen at upload
// (sniffed then), with nosniff; files are private to the caller's cache.
func Serve(w http.ResponseWriter, r *http.Request, deps httpx.Deps, stableID, path string) {
	f, ct, err := Open(stableID, path)
	switch {
	case errors.Is(err, ErrInvalidPath), errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "file not found")
		return
	case err != nil:
		if deps.Log != nil {
			deps.Log.Error("files: open", "err", err)
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not read the file")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not read the file")
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Disposition", "inline")
	// Files never change (random names), so the modification time is a fine validator.
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// QueryToken lets a GET request authenticate with ?access_token=<session token>
// when it has no Authorization header. React Native <Image> and Linking.openURL
// cannot always set headers.
//
// Tradeoff: a token in a URL can end up in server or proxy access logs and in
// browser history. It is the same full-access session token as the header, so
// use it only for file downloads (routes wrapped with this function), never log
// query strings, and prefer the Authorization header (RN <Image> supports
// source.headers). A future improvement is short-lived signed URLs.
func QueryToken(deps httpx.Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := auth.UserFrom(r.Context()); ok || r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			token := strings.TrimSpace(r.URL.Query().Get("access_token"))
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			r2 := r.Clone(r.Context())
			r2.Header.Set("Authorization", "Bearer "+token)
			auth.Middleware(deps)(next).ServeHTTP(w, r2)
		})
	}
}
