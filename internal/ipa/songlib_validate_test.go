package ipa

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveRejectsTraversal(t *testing.T) {
	for _, name := range []string{"../escape", "Payload/App.app/../../outside", "/absolute"} {
		p := filepath.Join(t.TempDir(), "bad.ipa")
		f, _ := os.Create(p)
		w := zip.NewWriter(f)
		entry, _ := w.Create(name)
		_, _ = entry.Write([]byte("test"))
		_ = w.Close()
		_ = f.Close()
		if ValidateArchive(p) == nil {
			t.Fatal(name)
		}
	}
}
