package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type payload struct {
	Name string `json:"name"`
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusNotFound, "not_found", "horse not found")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	want := `{"error":{"code":"not_found","message":"horse not found"}}`
	if body := strings.TrimSpace(rec.Body.String()); body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}

func TestReadJSON(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantOK   bool
		wantCode int
		wantErr  string
	}{
		{"valid", `{"name":"Luna"}`, true, http.StatusOK, ""},
		{"unknown field", `{"name":"Luna","x":1}`, false, http.StatusBadRequest, "invalid_json"},
		{"malformed", `{"name":`, false, http.StatusBadRequest, "invalid_json"},
		{"trailing data", `{"name":"Luna"} {}`, false, http.StatusBadRequest, "invalid_json"},
		{"too large", `{"name":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, false, http.StatusRequestEntityTooLarge, "body_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var p payload
			if ok := ReadJSON(rec, req, &p); ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (body %s)", ok, tt.wantOK, rec.Body)
			}
			if tt.wantOK {
				if p.Name != "Luna" {
					t.Errorf("name = %q", p.Name)
				}
				return
			}
			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if !strings.Contains(rec.Body.String(), `"code":"`+tt.wantErr+`"`) {
				t.Errorf("body = %s", rec.Body)
			}
		})
	}
}
