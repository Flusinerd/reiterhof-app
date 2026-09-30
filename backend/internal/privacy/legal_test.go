package privacy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
)

// The privacy text in docs/legal is the source of the app's text; a consent records
// privacy.TextVersion, so the constant and the text must change together.
func TestLegalTextsMatchTextVersion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "legal", "datenschutz.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "**Textversion:** "+privacy.TextVersion+"\n") {
		t.Errorf("datenschutz.md does not carry TextVersion %s", privacy.TextVersion)
	}
	if !strings.Contains(text, "Entwurf – vor Veröffentlichung rechtlich prüfen lassen") {
		t.Error("datenschutz.md lacks the draft marker")
	}
}
