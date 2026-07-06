//go:build !dev

package webspa

import (
	"io/fs"
	"testing"
)

func TestEmbeddedSPAIncludesNextExportAssets(t *testing.T) {
	nextStaticInfo, err := fs.Stat(SPA, "out/_next/static")
	if err != nil {
		t.Fatalf("embedded SPA missing Next static assets: %v", err)
	}
	if !nextStaticInfo.IsDir() {
		t.Fatalf("out/_next/static is not a directory")
	}

	if _, err := fs.Stat(SPA, "out/__next._tree.txt"); err != nil {
		t.Fatalf("embedded SPA missing Next RSC export data: %v", err)
	}
}
