//go:build !windows

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tarball(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestVerifyChecksumRejectsWhatTheChecksumFileDoesNotVouchFor(t *testing.T) {
	archive := tarball(t, map[string]string{releaseBin: "binary"})
	sum := sha256.Sum256(archive)
	good := hex.EncodeToString(sum[:])
	for name, c := range map[string]struct {
		sums string
		ok   bool
	}{
		"matching":        {sums: good + "  a.tar.gz\n", ok: true},
		"other file only": {sums: good + "  b.tar.gz\n"},
		"wrong digest":    {sums: strings.Repeat("0", 64) + "  a.tar.gz\n"},
		"empty":           {sums: ""},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "a.tar.gz"), archive, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "sums.txt"), []byte(c.sums), 0o600); err != nil {
				t.Fatal(err)
			}
			err := verifyChecksum(filepath.Join(dir, "a.tar.gz"), filepath.Join(dir, "sums.txt"))
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, ErrChecksum)) {
				t.Errorf("verifyChecksum = %v, want ok=%t", err, c.ok)
			}
		})
	}
}

func TestUnpackBinaryWritesOnlyTheNamedEntryToAFixedPath(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	archive := filepath.Join(src, "a.tar.gz")
	body := tarball(t, map[string]string{"../evil": "x", "README.md": "docs", "dir/" + releaseBin: "binary"})
	if err := os.WriteFile(archive, body, 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := unpackBinary(archive, dst)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dst, releaseBin) {
		t.Errorf("binary at %s", path)
	}
	if b, _ := os.ReadFile(path); string(b) != "binary" {
		t.Errorf("binary holds %q", b)
	}
	if entries, _ := os.ReadDir(dst); len(entries) != 1 {
		t.Errorf("unpack wrote %d entries, want 1", len(entries))
	}
	if _, err := os.Stat(filepath.Join(src, "evil")); err == nil {
		t.Error("an entry escaped the destination")
	}
}

func TestUnpackBinaryFailsOnAnArchiveWithoutTheBinary(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := os.WriteFile(archive, tarball(t, map[string]string{"README.md": "docs"}), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := unpackBinary(archive, t.TempDir()); err == nil {
		t.Fatal("unpacked an archive that holds no binary")
	}
}
