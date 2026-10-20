//go:build magiko

package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/magiko"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/testenv"
)

func TestMagikoHooksCoverNativeAndAPIRequests(t *testing.T) {
	home := t.TempDir()
	testenv.SetHome(t, home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	if err := magiko.Save(magiko.Preferences{Enabled: true, Text: "Use concise Chinese"}); err != nil {
		t.Fatal(err)
	}
	input := []byte(`{"model":"m","stream":true,"instructions":"native","input":"hi"}`)
	for _, path := range []string{"/v1/responses", CodexPath + "/responses"} {
		for _, remote := range []bool{false, true} {
			r := httptest.NewRequest("POST", "http://localhost"+path, bytes.NewReader(input))
			r.RemoteAddr = "127.0.0.1:45678"
			if remote {
				r.Header.Set("X-Forwarded-For", "192.0.2.10")
			}
			_, out, done := gatewayHooks(httptest.NewRecorder(), r, provider.Responses, input)
			done()
			if remote && !bytes.Equal(out, input) {
				t.Fatalf("preferences leaked to proxied client: %s", out)
			}
			if !remote && !bytes.Contains(out, []byte("Use concise Chinese")) {
				t.Fatalf("path %s did not apply preferences: %s", path, out)
			}
		}
	}
}

// Exercise the actual subscription HTTP route, not just its shared helper:
// the account headers, body fields and SSE reply must survive unchanged.
func TestMagikoNativeSubscriptionRoundTrip(t *testing.T) {
	setup(t, provider.Chat, &fake{t: t})
	type captured struct {
		body    []byte
		headers http.Header
	}
	got := make(chan captured, 1)
	chatgpt(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		got <- captured{body, r.Header.Clone()}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(`data: {"type":"response.completed","response":{"id":"magiko-fixture","usage":{"input_tokens":9,"output_tokens":2}}}`))
	})
	if err := magiko.Save(magiko.Preferences{Enabled: true, Text: "Preserve technical identifiers"}); err != nil {
		t.Fatal(err)
	}
	input := `{"model":"gpt-5.5","stream":true,"instructions":"existing","input":"hi","tools":[]}`
	req := httptest.NewRequest("POST", CodexPath+"/responses", strings.NewReader(input))
	req.RemoteAddr = "127.0.0.1:45678"
	req.Header.Set("Authorization", "Bearer fixture-chatgpt-token")
	req.Header.Set("chatgpt-account-id", "fixture-account")
	reply := httptest.NewRecorder()
	New().Handler().ServeHTTP(reply, req)
	if reply.Code != 200 || !strings.Contains(reply.Body.String(), "magiko-fixture") {
		t.Fatalf("native route: %d %s", reply.Code, reply.Body.String())
	}
	select {
	case call := <-got:
		if call.headers.Get("Authorization") != "Bearer fixture-chatgpt-token" || call.headers.Get("chatgpt-account-id") != "fixture-account" {
			t.Fatal("subscription headers changed")
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(call.body, &body); err != nil {
			t.Fatal(err)
		}
		var instructions string
		if err := json.Unmarshal(body["instructions"], &instructions); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(instructions, "existing") || !strings.Contains(instructions, "Preserve technical identifiers") {
			t.Fatalf("preferences missing: %s", instructions)
		}
		if string(body["stream"]) != "true" || string(body["model"]) != `"gpt-5.5"` || string(body["tools"]) != "[]" {
			t.Fatalf("changed model/stream/tools: %s", call.body)
		}
	default:
		t.Fatal("native upstream was not reached")
	}
}
