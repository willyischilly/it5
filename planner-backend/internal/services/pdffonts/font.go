package pdffonts

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed Arial.ttf
var arialTTF []byte

var (
	embeddedPath string
	embeddedOnce sync.Once
)

func ArialPath() string {
	embeddedOnce.Do(func() {
		if len(arialTTF) == 0 {
			return
		}
		f, err := os.CreateTemp("", "planner-arial-*.ttf")
		if err != nil {
			return
		}
		if _, err := f.Write(arialTTF); err != nil {
			_ = f.Close()
			return
		}
		_ = f.Close()
		embeddedPath = f.Name()
	})
	if embeddedPath != "" {
		return embeddedPath
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Join(filepath.Dir(file), "Arial.ttf")
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
