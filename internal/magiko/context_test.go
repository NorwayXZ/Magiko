package magiko

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/testenv"
)

func TestMain(m *testing.M) { os.Exit(testenv.Run(m)) }

func isolated(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	testenv.SetHome(t, home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	appdir.UseExecutable("")
	return home
}

func TestPreferencesPreserveRequests(t *testing.T) {
	p := Preferences{Enabled: true, Text: "Use Chinese; preserve code identifiers."}
	for _, tt := range []struct{ protocol, input, field string }{
		{"responses", `{"model":"m","stream":true,"instructions":"original","tools":[{"type":"function","name":"f"}],"input":"hi","future":{"x":1}}`, "instructions"},
		{"chat", `{"model":"m","stream":true,"tools":[],"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],"future":7}`, "messages"},
		{"anthropic", `{"model":"m","stream":true,"system":"original","messages":[],"future":7}`, "system"},
		{"anthropic", `{"system":[{"type":"text","text":"original","cache_control":{"type":"ephemeral"}}],"messages":[]}`, "system"},
	} {
		t.Run(tt.protocol+tt.input[:12], func(t *testing.T) {
			before := []byte(tt.input)
			out := Apply(tt.protocol, before, p)
			if bytes.Equal(before, out) || !bytes.Contains(out, []byte(marker)) {
				t.Fatalf("preferences missing: %s", out)
			}
			var a, b map[string]json.RawMessage
			json.Unmarshal(before, &a)
			json.Unmarshal(out, &b)
			for key, old := range a {
				if key != tt.field && !bytes.Equal(old, b[key]) {
					t.Fatalf("changed %s", key)
				}
			}
			if out2 := Apply(tt.protocol, out, p); !bytes.Equal(out, out2) {
				t.Fatal("duplicate preferences")
			}
			if tt.field != "messages" && !bytes.Contains(out, []byte("original")) {
				t.Fatal("lost original instructions")
			}
		})
	}
}

func TestPreferencesNoOpCases(t *testing.T) {
	for _, tt := range []struct {
		protocol, input string
		prefs           Preferences
	}{
		{"responses", `{"input":"hi"}`, Preferences{}},
		{"responses", `{"instructions":[]}`, Preferences{true, "normal"}},
		{"chat", `{"messages":null}`, Preferences{true, "normal"}},
		{"anthropic", `{"system":false}`, Preferences{true, "normal"}},
		{"gemini", `{"contents":[]}`, Preferences{true, "normal"}},
		{"responses", `null`, Preferences{true, "normal"}},
		{"responses", `not json`, Preferences{true, "normal"}},
		{"responses", `{}`, Preferences{true, strings.Repeat("a", MaxText+1)}},
		{"responses", `{}`, Preferences{true, "  "}},
	} {
		if out := Apply(tt.protocol, []byte(tt.input), tt.prefs); string(out) != tt.input {
			t.Fatalf("changed unsupported request: %s", out)
		}
	}
}

func TestPreferenceStorage(t *testing.T) {
	isolated(t)
	p, err := Load()
	if err != nil || p.Enabled {
		t.Fatalf("fresh state: %+v %v", p, err)
	}
	want := Preferences{true, "简体中文"}
	if err = Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || got != want {
		t.Fatalf("load: %+v %v", got, err)
	}
	if err = Save(Preferences{true, strings.Repeat("x", MaxText+1)}); err == nil {
		t.Fatal("oversize accepted")
	}
	got, _ = Load()
	if got != want {
		t.Fatal("invalid write changed state")
	}
	if err = os.WriteFile(preferencePath(), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(); err == nil {
		t.Fatal("corrupt preferences treated as valid")
	}
}

func TestSkillPackNeverOverwritesEdits(t *testing.T) {
	home := isolated(t)
	root, err := StageSkills()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(root, home+string(os.PathSeparator)) {
		t.Fatal("outside test home")
	}
	if _, err = StageSkills(); err != nil {
		t.Fatal(err)
	}
	for _, name := range Skills() {
		if _, err = os.Stat(filepath.Join(root, name, "LICENSE")); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(root, Skills()[0], "SKILL.md")
	if err = os.WriteFile(file, []byte("user edit"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = StageSkills(); err == nil {
		t.Fatal("edited pack accepted")
	}
	b, _ := os.ReadFile(file)
	if string(b) != "user edit" {
		t.Fatal("user edit overwritten")
	}
}

func TestSkillPackRefusesSymlink(t *testing.T) {
	isolated(t)
	root, err := StageSkills()
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, Skills()[0], "SKILL.md")
	original, _ := os.ReadFile(file)
	target := filepath.Join(t.TempDir(), "private")
	os.WriteFile(target, original, 0600)
	os.Remove(file)
	if err = os.Symlink(target, file); err != nil {
		t.Skip(err)
	}
	if _, err = StageSkills(); err == nil {
		t.Fatal("symlink accepted")
	}
}
