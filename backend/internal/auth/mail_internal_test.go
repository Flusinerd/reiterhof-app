package auth

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

// readParts returns the leaf parts of a multipart body as content type -> decoded content,
// plus the headers of each leaf.
func readParts(t *testing.T, contentType string, body io.Reader) (map[string]string, map[string]multipart.Part) {
	t.Helper()
	mt, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mt, "multipart/") {
		t.Fatalf("content type %q: %v", contentType, err)
	}
	out := map[string]string{}
	heads := map[string]multipart.Part{}
	r := multipart.NewReader(body, params["boundary"])
	for {
		p, err := r.NextRawPart()
		if err == io.EOF {
			return out, heads
		}
		if err != nil {
			t.Fatal(err)
		}
		ct := p.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/") {
			sub, subHeads := readParts(t, ct, p)
			for k, v := range sub {
				out[k] = v
				heads[k] = subHeads[k]
			}
			continue
		}
		var data []byte
		switch p.Header.Get("Content-Transfer-Encoding") {
		case "quoted-printable":
			data, err = io.ReadAll(quotedprintable.NewReader(p))
		default:
			data, err = io.ReadAll(p)
		}
		if err != nil {
			t.Fatal(err)
		}
		mt, _, _ := mime.ParseMediaType(ct)
		out[mt] = string(data)
		heads[mt] = *p
	}
}

func TestBuildMessageBranded(t *testing.T) {
	from := &mail.Address{Name: "Stallfunk", Address: "login@stallfunk.de"}
	to := &mail.Address{Address: "jan@example.org"}
	msg, err := loginMail("jan@example.org", "012345", "tok_EN-1", "https://api.example.org").Message(to.Address, "Dein Stallfunk-Code: 012345")
	if err != nil {
		t.Fatal(err)
	}
	raw := buildMessage(from, to, msg)
	for i, line := range bytes.Split(raw, []byte("\r\n")) {
		if len(line) > 998 {
			t.Fatalf("line %d is %d bytes long, over the SMTP limit", i, len(line))
		}
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	parts, heads := readParts(t, m.Header.Get("Content-Type"), m.Body)

	text := parts["text/plain"]
	for _, want := range []string{"012 345", "stallfunk://auth/verify?token=tok_EN-1", "https://api.example.org/auth/verify?token=tok_EN-1", "15 Minuten"} {
		if !strings.Contains(text, want) {
			t.Errorf("text part lacks %q:\n%s", want, text)
		}
	}
	html := parts["text/html"]
	for _, want := range []string{"012 345", `href="https://api.example.org/auth/verify?token=tok_EN-1"`, `src="cid:` + mailLogoCID + `"`, "Dein Anmeldecode"} {
		if !strings.Contains(html, want) {
			t.Errorf("html part lacks %q", want)
		}
	}
	if strings.Contains(html, "stallfunk://") {
		t.Error("html part carries the custom scheme link")
	}
	logo, ok := heads["image/png"]
	if !ok || logo.Header.Get("Content-ID") != "<"+mailLogoCID+">" || !strings.HasPrefix(parts["image/png"], "iVBORw0KGgo") {
		t.Fatalf("logo part missing or wrong: %v", logo.Header)
	}
}

func TestMailContentEscapesAndDropsNonHTTPS(t *testing.T) {
	msg, err := MailContent{
		Heading:   "Hallo <b>Welt</b>",
		Intro:     []string{`"quoted" & <script>alert(1)</script>`},
		Button:    "Los",
		ButtonURL: "javascript:alert(1)",
	}.Message("a@example.org", "x")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(msg.HTML, "<script>") || strings.Contains(msg.HTML, "<b>Welt") {
		t.Fatal("html is not escaped")
	}
	if strings.Contains(msg.HTML, "javascript:") || strings.Contains(msg.HTML, ">Los</a>") {
		t.Fatal("a non-https button was rendered")
	}
	if !strings.Contains(msg.Body, "Hallo <b>Welt</b>") {
		t.Fatalf("text part = %q", msg.Body)
	}
}

func TestBuildMessagePlain(t *testing.T) {
	raw := buildMessage(&mail.Address{Address: "a@example.org"}, &mail.Address{Address: "b@example.org"},
		Message{Subject: "Grüße", Body: "Zeile 1\nZeile 2"})
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if ct := m.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content type %q", ct)
	}
	body, _ := io.ReadAll(quotedprintable.NewReader(m.Body))
	if !strings.Contains(string(body), "Zeile 1\r\nZeile 2") {
		t.Fatalf("body = %q", body)
	}
}
