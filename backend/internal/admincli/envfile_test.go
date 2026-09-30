package admincli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/admincli"
)

func TestParseEnvFile(t *testing.T) {
	const in = `# Environment of the Stallfunk API
; another comment style

REITERHOF_ADDR=127.0.0.1:8080
  REITERHOF_DATABASE_URL = postgres://reiterhof:secret@127.0.0.1:5432/reiterhof?sslmode=disable
#REITERHOF_SMTP_HOST=commented.example
REITERHOF_SMTP_FROM="Stallfunk <login@stallfunk.de>"
REITERHOF_SMTP_USER='re # not a comment'
REITERHOF_SMTP_PASSWORD="pa\"ss\\word"
EMPTY=
EMPTY_QUOTED=""
SPACED="  keep me  "
HASH_IN_VALUE=abc#def
DUP=first
DUP=second
LONG=one \
  two
`
	got, err := admincli.ParseEnvFile(in)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"REITERHOF_ADDR":          "127.0.0.1:8080",
		"REITERHOF_DATABASE_URL":  "postgres://reiterhof:secret@127.0.0.1:5432/reiterhof?sslmode=disable",
		"REITERHOF_SMTP_FROM":     "Stallfunk <login@stallfunk.de>",
		"REITERHOF_SMTP_USER":     "re # not a comment",
		"REITERHOF_SMTP_PASSWORD": `pa"ss\word`,
		"EMPTY":                   "",
		"EMPTY_QUOTED":            "",
		"SPACED":                  "  keep me  ",
		"HASH_IN_VALUE":           "abc#def",
		"DUP":                     "second",
		"LONG":                    "one two",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseEnvFile mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseEnvFileCRLF(t *testing.T) {
	got, err := admincli.ParseEnvFile("A=1\r\nB=\"two\"\r\n")
	if err != nil || got["A"] != "1" || got["B"] != "two" {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestParseEnvFileErrors(t *testing.T) {
	for name, in := range map[string]string{
		"no equals":          "JUSTAKEY",
		"bad key":            "1BAD=x",
		"key with dash":      "A-B=x",
		"empty key":          "=x",
		"unterminated dq":    `A="open`,
		"unterminated sq":    `A='open`,
		"text after quote":   `A="x" y`,
		"text after squote":  `A='x' y`,
		"trailing backslash": `A="x\`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := admincli.ParseEnvFile("OK=1\n" + in + "\n")
			if err == nil {
				t.Fatal("want error")
			}
			if !strings.Contains(err.Error(), "line 2") {
				t.Errorf("error %q should name line 2", err)
			}
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.env")
	if err := os.WriteFile(path, []byte("ADMINCLI_TEST_NEW=from-file\nADMINCLI_TEST_SET=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ADMINCLI_TEST_SET", "from-env")
	t.Setenv("ADMINCLI_TEST_NEW", "") // registers the cleanup; unset it for the test
	if err := os.Unsetenv("ADMINCLI_TEST_NEW"); err != nil {
		t.Fatal(err)
	}

	set, err := admincli.LoadEnvFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(set, []string{"ADMINCLI_TEST_NEW"}) {
		t.Errorf("set = %v, want only the variable that was unset", set)
	}
	if got := os.Getenv("ADMINCLI_TEST_NEW"); got != "from-file" {
		t.Errorf("ADMINCLI_TEST_NEW = %q", got)
	}
	if got := os.Getenv("ADMINCLI_TEST_SET"); got != "from-env" {
		t.Errorf("environment must win, ADMINCLI_TEST_SET = %q", got)
	}
}

func TestLoadEnvFileMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.env")
	if set, err := admincli.LoadEnvFile(missing, true); err != nil || set != nil {
		t.Errorf("optional missing file: %v, %v", set, err)
	}
	if _, err := admincli.LoadEnvFile(missing, false); err == nil {
		t.Error("required missing file must fail")
	}
}

func TestLoadEnvFileSyntaxError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.env")
	if err := os.WriteFile(path, []byte("garbage\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := admincli.LoadEnvFile(path, false)
	if err == nil || !strings.Contains(err.Error(), "bad.env") || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("err = %v, want file name and line", err)
	}
}
