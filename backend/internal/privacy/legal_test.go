package privacy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
)

// The legal texts in docs/legal are the source of the app's texts; a consent records
// privacy.TextVersion, so the constant and the texts must change together.
func TestLegalTextsMatchTextVersion(t *testing.T) {
	for _, name := range []string{"datenschutz.md", "impressum.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "legal", name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, "**Textversion:** "+privacy.TextVersion+"\n") {
			t.Errorf("%s does not carry TextVersion %s", name, privacy.TextVersion)
		}
		if !strings.Contains(text, "Entwurf – vor Veröffentlichung rechtlich prüfen lassen") {
			t.Errorf("%s lacks the draft marker", name)
		}
	}
}
