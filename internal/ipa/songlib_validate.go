package ipa

import (
	"archive/zip"
	"fmt"
	"path"
	"strings"
)

func ValidateArchive(filename string) error {
	z, e := zip.OpenReader(filename)
	if e != nil {
		return e
	}
	defer func() { _ = z.Close() }()
	if len(z.File) > 20000 {
		return fmt.Errorf("too many IPA entries")
	}
	var total uint64
	for _, f := range z.File {
		n := f.Name
		if strings.Contains(n, "\\") || strings.HasPrefix(n, "/") || path.Clean(n) == ".." || strings.HasPrefix(path.Clean(n), "../") || strings.Contains(n, "/../") {
			return fmt.Errorf("unsafe IPA path")
		}
		if f.UncompressedSize64 > 512<<20 {
			return fmt.Errorf("IPA entry exceeds size limit")
		}
		total += f.UncompressedSize64
		if total > 2<<30 {
			return fmt.Errorf("expanded IPA exceeds size limit")
		}
		if f.CompressedSize64 > 0 && f.UncompressedSize64/f.CompressedSize64 > 1000 {
			return fmt.Errorf("suspicious IPA compression ratio")
		}
	}
	return nil
}
