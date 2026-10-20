//go:build !magiko

// Package brand contains build-time application identity. The magiko tag
// isolates the fork without changing upstream's test and integration contracts.
package brand

const (
	Name       = "magpie"
	Slug       = "magpie"
	BundleID   = "com.yetone.magpie"
	InstanceID = "ai.usemagpie.app.i"
	Port       = 3425
	Fork       = false
)
