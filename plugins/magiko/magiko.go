// Package magiko registers the fork's workspace, using the shared window kernel.
package magiko

import (
	"github.com/yetone/magpie/internal/brand"
	"github.com/yetone/magpie/internal/pluginrt"
)

func init() {
	if !brand.Fork {
		return
	}
	pluginrt.Register(pluginrt.Manifest{
		ID: "magiko", API: "^1", Requires: []string{"window", "gateway"}, Essential: true,
		Name: "Magiko workspace", Sub: "Local request preferences and optional reverse-engineering skills",
		Parts: []pluginrt.Part{{Slot: "nav.page", Key: "magiko"}}, Client: []string{"client.js"},
	})
}
