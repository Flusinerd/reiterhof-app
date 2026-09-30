package files

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Register adds the upload and download routes.
//
//	POST /api/v1/files               multipart/form-data, field "file" -> 201 Saved
//	POST /api/v1/files/download-link {url} -> {url, expires_at}, see links.go
//	GET  /api/v1/files/{path...}     download; only files a record of the stable references
//
// Errors: 400 invalid_upload (also for a damaged image), 413 file_too_large, 415 unsupported_type,
// 404 not_found (also for paths of other stables, malformed paths and files nothing points to).
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	mux.Handle("POST /api/v1/files", auth.RequireStable(http.HandlerFunc(h.upload)))
	mux.Handle("POST /api/v1/files/download-link", auth.RequireStable(http.HandlerFunc(h.downloadLink)))
	mux.Handle("GET /api/v1/files/{path...}", DownloadLink(deps)(auth.RequireStable(http.HandlerFunc(h.download))))
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
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_type", "only JPEG, PNG, WebP and PDF are allowed")
	case errors.Is(err, ErrCorrupt):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_upload", "the image file is damaged")
	default:
		if deps.Log != nil {
			deps.Log.Error("files: upload", "err", err)
		}
		httpx.WriteError(w, http.StatusBadRequest, "invalid_upload", "could not read the upload")
	}
}

// download serves a file by its path with the visibility of the record that references it
// (least privilege): a horse document only to the owner, the riders and admins of the horse;
// an observation photo or a blanket photo to every member of the stable. A path no record
// points to (never attached, or detached since) is a 404 for everybody, like a path of
// another stable.
func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	path := r.PathValue("path")
	if !Belongs(user.StableID, path) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	ref, err := referenceOf(r.Context(), h.deps.Pool, user.StableID, path)
	if err != nil {
		if h.deps.Log != nil {
			h.deps.Log.Error("files: reference lookup", "err", err)
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not read the file")
		return
	}
	allowed := false
	switch ref.kind {
	case refDocument:
		allowed, err = auth.CanManageHorse(r.Context(), h.deps.Pool, user, ref.horseID)
		if err == nil && !allowed {
			allowed, err = auth.IsRider(r.Context(), h.deps.Pool, user, ref.horseID)
		}
		if err != nil {
			if h.deps.Log != nil {
				h.deps.Log.Error("files: document access", "err", err)
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not read the file")
			return
		}
	case refObservation, refBlanket:
		allowed = true
	}
	if !allowed {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	Serve(w, r, h.deps, user.StableID, path)
}

type refKind int

const (
	refNone refKind = iota
	refDocument
	refObservation
	refBlanket
)

type reference struct {
	kind    refKind
	horseID string
}

// referenceOf finds the record of the stable that points to path. A document wins over
// any other reference to the same file (the stricter rule applies).
func referenceOf(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, stableID, path string) (reference, error) {
	var (
		kind    int
		horseID string
	)
	err := q.QueryRow(ctx, `SELECT kind, horse_id::text FROM (
			SELECT 1 AS kind, horse_id FROM horse_documents WHERE stable_id = $1 AND file_path = $2
			UNION ALL SELECT 2, horse_id FROM observations WHERE stable_id = $1 AND media ? $2
			UNION ALL SELECT 3, horse_id FROM blankets WHERE stable_id = $1 AND photo_path = $2
		) r ORDER BY kind LIMIT 1`, stableID, path).Scan(&kind, &horseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return reference{}, nil
	}
	if err != nil {
		return reference{}, err
	}
	return reference{kind: refKind(kind), horseID: horseID}, nil
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
