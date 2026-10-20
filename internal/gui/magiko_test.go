//go:build magiko

package gui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/magiko"
	"github.com/yetone/magpie/internal/testenv"
)

func TestMagikoRoutesRoundTripAndImport(t *testing.T) {
	home := t.TempDir()
	testenv.SetHome(t, home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	appdir.UseExecutable("")
	// Simulate an existing Codex with sign-in: import must leave both intact.
	codex := filepath.Join(home, ".codex")
	os.MkdirAll(codex, 0700)
	auth := []byte(`{"tokens":{"access_token":"fixture-not-real"}}`)
	config := []byte("model = \"fixture-model\"\n")
	os.WriteFile(filepath.Join(codex, "auth.json"), auth, 0600)
	os.WriteFile(filepath.Join(codex, "config.toml"), config, 0600)
	mux := http.NewServeMux()
	magikoRoutes(mux)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	if w := call("GET", "/api/magiko", ""); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	for _, body := range []string{`{"enabled":true,"text":"Concise Chinese"}`, `{"enabled":false,"text":""}`} {
		if w := call("POST", "/api/magiko/preferences", body); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	for _, body := range []string{`{"enabled":true,"text":""}`, `{"unknown":1}`, `{} {}`, `not-json`, `null`, strings.Repeat("x", magiko.MaxText*6+1025)} {
		if w := call("POST", "/api/magiko/preferences", body); w.Code != 400 {
			t.Fatalf("accepted malformed config: %d", w.Code)
		}
	}
	for range 2 {
		if w := call("POST", "/api/magiko/skills", `{}`); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	view, err := library.Read(nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(view)
	if !bytes.Contains(data, []byte("protocol-reverse-engineering")) {
		t.Fatalf("pack not imported: %s", data)
	}
	for _, name := range magiko.Skills() {
		if _, err := os.Lstat(filepath.Join(codex, "skills", name)); !os.IsNotExist(err) {
			t.Fatalf("skill %s deployed without consent", name)
		}
	}
	for file, want := range map[string][]byte{"auth.json": auth, "config.toml": config} {
		got, err := os.ReadFile(filepath.Join(codex, file))
		if err != nil || !bytes.Equal(want, got) {
			t.Fatalf("changed Codex %s", file)
		}
	}
}

func TestMagikoVersionedPageBrand(t *testing.T) {
	b := versionedPage([]byte(`<title>magpie</title><span>magpie</span><script src="app.js"></script>`))
	if !bytes.Contains(b, []byte("<title>Magiko</title>")) || !bytes.Contains(b, []byte("app.js?v=")) {
		t.Fatalf("branding/cache contract: %s", b)
	}
}
