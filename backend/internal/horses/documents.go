package horses

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// DocumentKinds are the accepted values of horse_documents.kind.
var DocumentKinds = []string{"passport", "vaccination_record", "insurance", "other"}

// Document is a horse document (JAN-53). URL serves the file with the role check of the horse.
type Document struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	ContentType string    `json:"content_type"`
	UploadedBy  *Person   `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *handler) registerDocuments(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/horses/{id}/documents", auth.RequireStable(http.HandlerFunc(h.listDocuments)))
	mux.Handle("POST /api/v1/horses/{id}/documents", auth.RequireStable(http.HandlerFunc(h.createDocument)))
	mux.Handle("PATCH /api/v1/horses/{id}/documents/{docId}", auth.RequireStable(http.HandlerFunc(h.patchDocument)))
	mux.Handle("DELETE /api/v1/horses/{id}/documents/{docId}", auth.RequireStable(http.HandlerFunc(h.deleteDocument)))
	mux.Handle("GET /api/v1/horses/{id}/documents/{docId}/file",
		files.DownloadLink(h.deps)(auth.RequireStable(http.HandlerFunc(h.documentFile))))
}

// canRead answers 404 for horses outside the stable and 403 unless the caller is the owner,
// an admin or a rider of the horse. Documents (passport, insurance) are private to them.
func (h *handler) canReadDocuments(w http.ResponseWriter, r *http.Request) (auth.User, string, bool) {
	user, id, ok := h.horseInStable(w, r)
	if !ok {
		return user, id, false
	}
	manage, err := auth.CanManageHorse(r.Context(), h.deps.Pool, user, id)
	if err == nil && !manage {
		manage, err = auth.IsRider(r.Context(), h.deps.Pool, user, id)
	}
	if err != nil {
		h.fail(w, "document access", err)
		return user, id, false
	}
	if !manage {
		forbidden(w, "documents are visible to the owner, riders and admins")
		return user, id, false
	}
	return user, id, true
}

func documentURL(horseID, docID string) string {
	return "/api/v1/horses/" + horseID + "/documents/" + docID + "/file"
}

func (h *handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canReadDocuments(w, r)
	if !ok {
		return
	}
	rows, err := h.deps.Pool.Query(r.Context(), `SELECT d.id, d.kind, d.title, d.file_path, d.created_at,
			u.id, u.name, u.avatar_color
		FROM horse_documents d LEFT JOIN users u ON u.id = d.uploaded_by
		WHERE d.horse_id = $1 AND d.stable_id = $2 ORDER BY d.created_at DESC, d.id`, id, user.StableID)
	if err != nil {
		h.fail(w, "documents", err)
		return
	}
	defer rows.Close()
	docs := []Document{}
	for rows.Next() {
		var (
			d    Document
			path string
			uid  *string
			un   *string
			uc   *string
		)
		if err := rows.Scan(&d.ID, &d.Kind, &d.Title, &path, &d.CreatedAt, &uid, &un, &uc); err != nil {
			h.fail(w, "documents scan", err)
			return
		}
		d.URL = documentURL(id, d.ID)
		d.ContentType, _ = files.ContentTypeOf(path)
		if uid != nil && un != nil {
			d.UploadedBy = &Person{ID: *uid, Name: *un, ColorKey: uc}
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, "documents", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, docs)
}

func validTitle(t string) (string, bool) {
	t = strings.TrimSpace(t)
	return t, t != "" && utf8.RuneCountInString(t) <= 120
}

func (h *handler) createDocument(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var in struct {
		Kind     string `json:"kind"`
		Title    string `json:"title"`
		FilePath string `json:"file_path"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	title, okTitle := validTitle(in.Title)
	if !okTitle {
		invalid(w, "title must be 1 to 120 characters")
		return
	}
	if !contains(DocumentKinds, in.Kind) {
		invalid(w, "kind must be one of "+strings.Join(DocumentKinds, ", "))
		return
	}
	// The path must come from the files API for this stable and the file must exist.
	f, _, err := files.Open(user.StableID, in.FilePath)
	if err != nil {
		invalid(w, "file_path must be the path of a file uploaded to this stable")
		return
	}
	_ = f.Close()

	var d Document
	err = h.deps.Pool.QueryRow(r.Context(), `INSERT INTO horse_documents (stable_id, horse_id, kind, title, file_path, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		user.StableID, id, in.Kind, title, in.FilePath, user.ID).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		h.fail(w, "create document", err)
		return
	}
	d.Kind, d.Title, d.URL = in.Kind, title, documentURL(id, d.ID)
	d.ContentType, _ = files.ContentTypeOf(in.FilePath)
	d.UploadedBy = &Person{ID: user.ID, Name: user.Name}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

func (h *handler) patchDocument(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var in struct {
		Kind  *string `json:"kind"`
		Title *string `json:"title"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Title != nil {
		t, ok := validTitle(*in.Title)
		if !ok {
			invalid(w, "title must be 1 to 120 characters")
			return
		}
		in.Title = &t
	}
	if in.Kind != nil && !contains(DocumentKinds, *in.Kind) {
		invalid(w, "kind must be one of "+strings.Join(DocumentKinds, ", "))
		return
	}
	var d Document
	var path string
	err := h.deps.Pool.QueryRow(r.Context(), `UPDATE horse_documents SET kind = COALESCE($4, kind), title = COALESCE($5, title)
		WHERE id = $1 AND horse_id = $2 AND stable_id = $3
		RETURNING id, kind, title, file_path, created_at`,
		r.PathValue("docId"), id, user.StableID, in.Kind, in.Title).
		Scan(&d.ID, &d.Kind, &d.Title, &path, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "document not found")
		return
	}
	if err != nil {
		h.fail(w, "patch document", err)
		return
	}
	d.URL = documentURL(id, d.ID)
	d.ContentType, _ = files.ContentTypeOf(path)
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var path string
	err := h.deps.Pool.QueryRow(r.Context(), `DELETE FROM horse_documents WHERE id = $1 AND horse_id = $2 AND stable_id = $3
		RETURNING file_path`, r.PathValue("docId"), id, user.StableID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "document not found")
		return
	}
	if err != nil {
		h.fail(w, "delete document", err)
		return
	}
	// Best effort: an orphaned file is harmless, a failed request is not.
	if err := files.Remove(user.StableID, path); err != nil {
		h.deps.Log.Warn("horses: remove document file", "path", path, "err", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) documentFile(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canReadDocuments(w, r)
	if !ok {
		return
	}
	var path string
	err := h.deps.Pool.QueryRow(r.Context(), `SELECT file_path FROM horse_documents
		WHERE id = $1 AND horse_id = $2 AND stable_id = $3`, r.PathValue("docId"), id, user.StableID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "document not found")
		return
	}
	if err != nil {
		h.fail(w, "document file", err)
		return
	}
	files.Serve(w, r, h.deps, user.StableID, path)
}
