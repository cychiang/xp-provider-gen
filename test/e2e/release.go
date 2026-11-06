//go:build !windows

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	releaseRepo  = "cychiang/xp-provider-gen"
	releaseBin   = "xp-provider-gen"
	maxBinaryLen = 256 << 20
)

var ErrChecksum = errors.New("checksum mismatch")

// newestReleases asks GitHub for the newest published release tags, newest first.
func newestReleases(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, "gh", "release", "list", "-R", releaseRepo, "--limit", "1",
		"--json", "tagName", "--jq", ".[].tagName").Output() // #nosec G204 -- fixed command, no shell
	if err != nil {
		return nil, fmt.Errorf("listing releases with gh: %w", err)
	}
	return strings.Fields(string(out)), nil
}

// fetchRelease downloads a release's archive for this host into a new empty
// directory under dest, verifies it against the release's checksum file, and
// only then unpacks the binary into a second new directory. It returns the
// binary's path.
func fetchRelease(ctx context.Context, tag, dest string) (string, error) {
	version := strings.TrimPrefix(tag, "v")
	archive := fmt.Sprintf("%s_%s_%s_%s.tar.gz", releaseBin, version, runtime.GOOS, runtime.GOARCH)
	sums := fmt.Sprintf("%s_%s_checksums.txt", releaseBin, version)
	download, err := os.MkdirTemp(dest, "download-")
	if err != nil {
		return "", err
	}
	// #nosec G204 -- the arguments are the declared tag and fixed names, no shell
	cmd := exec.CommandContext(ctx, "gh", "release", "download", tag, "-R", releaseRepo,
		"-p", archive, "-p", sums, "-D", download)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("downloading %s: %w\n%s", tag, err, out)
	}
	if err := verifyChecksum(filepath.Join(download, archive), filepath.Join(download, sums)); err != nil {
		return "", err
	}
	bin, err := os.MkdirTemp(dest, "bin-")
	if err != nil {
		return "", err
	}
	return unpackBinary(filepath.Join(download, archive), bin)
}

// verifyChecksum fails unless the archive's sha256 is the one its name has in
// the checksum file.
func verifyChecksum(archive, sums string) error {
	list, err := os.ReadFile(sums) // #nosec G304 -- a path in the private work directory
	if err != nil {
		return err
	}
	var want string
	for _, line := range strings.Split(string(list), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == filepath.Base(archive) {
			want = f[0]
		}
	}
	if want == "" {
		return fmt.Errorf("%w: %s lists no %s", ErrChecksum, filepath.Base(sums), filepath.Base(archive))
	}
	b, err := os.ReadFile(archive) // #nosec G304 -- a path in the private work directory
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("%w for %s: archive is %s, checksum file says %s", ErrChecksum, filepath.Base(archive), got, want)
	}
	return nil
}

// unpackBinary writes the archive's one executable into dir. The archive is
// untrusted: only the entry named releaseBin is read, under a size cap, and it
// is written to a fixed path.
func unpackBinary(archive, dir string) (string, error) {
	b, err := os.ReadFile(archive) // #nosec G304 -- a path in the private work directory
	if err != nil {
		return "", err
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%s holds no %s", filepath.Base(archive), releaseBin)
		}
		if err != nil {
			return "", err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == releaseBin {
			return writeBinary(tr, filepath.Join(dir, releaseBin))
		}
	}
}

func writeBinary(r io.Reader, path string) (string, error) {
	// #nosec G302 G304 -- an executable at a fixed path in the private work directory
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxBinaryLen+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxBinaryLen {
		err = fmt.Errorf("%s is larger than %d bytes", releaseBin, maxBinaryLen)
	}
	return path, err
}
