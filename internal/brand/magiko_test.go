//go:build magiko

package brand_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/brand"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/testenv"
	"github.com/yetone/magpie/internal/update"
)

func TestMain(m *testing.M) { os.Exit(testenv.Run(m)) }

func TestMagikoIdentityAndManualUpdates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	appdir.UseExecutable("")
	if !brand.Fork || brand.Name != "Magiko" || brand.Port != 3525 || settings.DefaultPort != 3525 {
		t.Fatal("wrong build identity")
	}
	if got := appdir.Config(); got != filepath.Join(home, "config", "magiko") {
		t.Fatalf("config collision: %s", got)
	}
	if got := appdir.Cache(); got != filepath.Join(home, "cache", "magiko") {
		t.Fatalf("cache collision: %s", got)
	}
	if update.Released("0.1.0-preview.1") || update.Released("1.0.0") {
		t.Fatal("upstream autoupdate enabled")
	}
	// An invalid, deliberately unusable upstream feed must never be consulted.
	t.Setenv("MAGPIE_UPDATE_FEED", ":invalid")
	if _, err := update.Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "manual") {
		t.Fatalf("unexpected update path: %v", err)
	}
}
