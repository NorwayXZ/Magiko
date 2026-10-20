package gui

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/brand"
	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/magiko"
)

// Registered inside Handler: the desktop and web shells retain the same
// authentication, host and same-origin protections as the existing APIs.
func magikoRoutes(mux *http.ServeMux) {
	if !brand.Fork {
		return
	}
	mux.HandleFunc("GET /api/magiko", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		prefs, err := magiko.Load()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"preferences": prefs, "skills": magiko.Skills(), "configDir": appdir.Config(), "maxText": magiko.MaxText})
	})
	mux.HandleFunc("POST /api/magiko/preferences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var prefs *magiko.Preferences
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, magiko.MaxText*6+1024))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&prefs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if prefs == nil {
			http.Error(w, "expected a JSON object", http.StatusBadRequest)
			return
		}
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			http.Error(w, "expected one JSON object", http.StatusBadRequest)
			return
		}
		if err := magiko.Save(*prefs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, prefs)
	})
	mux.HandleFunc("POST /api/magiko/skills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		dir, err := magiko.StageSkills()
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		probe, err := library.ProbeSkills(dir)
		if err != nil {
			fail(w, err)
			return
		}
		paths := make([]string, 0, len(probe.Candidates))
		for _, candidate := range probe.Candidates {
			paths = append(paths, candidate.Path)
		}
		// Import into the existing Library with no agents selected. Activation,
		// removal and per-agent assignment stay in its established backup/sync UI.
		result, err := library.InstallSkills(dir, paths, []string{})
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, result)
	})
}
