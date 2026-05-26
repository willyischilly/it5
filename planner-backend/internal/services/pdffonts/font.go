package pdffonts

import (
	"os"
	"path/filepath"
	"runtime"
)

func ArialPath() string {
	var base string
	if _, file, _, ok := runtime.Caller(0); ok {
		base = filepath.Join(filepath.Dir(file), "Arial.ttf")
		if _, err := os.Stat(base); err == nil {
			return base
		}
	}
	candidates := []string{
		"internal/services/pdffonts/Arial.ttf",
		filepath.Join("planner-backend", "internal/services/pdffonts/Arial.ttf"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
