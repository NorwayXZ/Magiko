// Package magiko implements explicit, optional request preferences. It does
// not change credentials, providers, model selection or model responses.
package magiko

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/edit"
)

const MaxText = 12000
const marker = "[Magiko preferences]"

// Preferences apply only to local agent requests; sharing the gateway never
// shares the owner's personal preferences with remote API clients.
type Preferences struct {
	Enabled bool   `json:"enabled"`
	Text    string `json:"text"`
}

var mu sync.Mutex

func preferencePath() string { return filepath.Join(appdir.Config(), "magiko-preferences.json") }

func Load() (Preferences, error) {
	mu.Lock()
	defer mu.Unlock()
	var p Preferences
	f, err := os.Open(preferencePath())
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, MaxText*6+1024))
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(b, &p); err != nil {
		return Preferences{}, err
	}
	if err = validate(p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

func validate(p Preferences) error {
	if !utf8.ValidString(p.Text) || len(p.Text) > MaxText {
		return fmt.Errorf("preferences must be valid UTF-8 and at most %d bytes", MaxText)
	}
	if p.Enabled && strings.TrimSpace(p.Text) == "" {
		return errors.New("enter preferences before enabling them")
	}
	return nil
}

// Save only writes Magiko's own file, never an agent's config or auth file.
func Save(p Preferences) error {
	if err := validate(p); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	path := preferencePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return edit.WriteAtomic(path, append(b, '\n'))
}

// Apply is deliberately conservative: malformed or unsupported bodies remain
// byte-for-byte intact, including future instruction shapes. Existing fields
// are retained as raw JSON; only the protocol's context field is changed.
func Apply(protocol string, body []byte, p Preferences) []byte {
	if !p.Enabled || validate(p) != nil {
		return body
	}
	text := marker + "\n" + strings.TrimSpace(p.Text)
	var obj map[string]json.RawMessage
	if json.Unmarshal(body, &obj) != nil || obj == nil {
		return body
	}
	put := func(key string, value any) { obj[key], _ = json.Marshal(value) }
	switch protocol {
	case "responses":
		var old string
		if raw := obj["instructions"]; len(raw) != 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			if json.Unmarshal(raw, &old) != nil {
				return body
			}
		}
		if strings.Contains(old, text) {
			return body
		}
		if old != "" {
			text = old + "\n\n" + text
		}
		put("instructions", text)
	case "chat":
		var messages []json.RawMessage
		if json.Unmarshal(obj["messages"], &messages) != nil || messages == nil {
			return body
		}
		for _, raw := range messages {
			var m struct {
				Role    string
				Content string
			}
			if json.Unmarshal(raw, &m) == nil && m.Role == "system" && m.Content == text {
				return body
			}
		}
		raw, _ := json.Marshal(map[string]string{"role": "system", "content": text})
		put("messages", append([]json.RawMessage{raw}, messages...))
	case "anthropic":
		var old string
		if raw := obj["system"]; len(raw) == 0 || json.Unmarshal(raw, &old) == nil {
			if strings.Contains(old, text) {
				return body
			}
			if old != "" {
				text = old + "\n\n" + text
			}
			put("system", text)
		} else {
			var blocks []json.RawMessage
			if json.Unmarshal(raw, &blocks) != nil || blocks == nil {
				return body
			}
			for _, raw := range blocks {
				var block struct {
					Type string
					Text string
				}
				if json.Unmarshal(raw, &block) == nil && block.Type == "text" && block.Text == text {
					return body
				}
			}
			block, _ := json.Marshal(map[string]string{"type": "text", "text": text})
			put("system", append(blocks, block))
		}
	default:
		return body
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return body
	}
	return out
}
