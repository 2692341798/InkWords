package bookrender

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	exportdomain "inkwords-backend/services/export-service/domain/export"
)

const fontListFormat = "%{postscriptname}\t%{family}\t%{file}\t%{index}\t%{fontversion}\n"

// Do not embed bytes.Buffer: its promoted ReadFrom bypasses Write's limit
// when os/exec drains stdout with io.Copy.
type fontListOutput struct{ buffer bytes.Buffer }

func (b *fontListOutput) Len() int       { return b.buffer.Len() }
func (b *fontListOutput) Bytes() []byte  { return b.buffer.Bytes() }
func (b *fontListOutput) String() string { return b.buffer.String() }

func (b *fontListOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 1<<20 {
		return 0, fmt.Errorf("font inventory output exceeds budget")
	}
	return b.buffer.Write(p)
}

func installedFontFaces(ctx context.Context) ([]exportdomain.FontFileEvidence, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "fc-list", "--format", fontListFormat)
	var output fontListOutput
	command.Stdout = &output
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("font inventory unavailable")
	}
	return parseFontInventory(output.String(), hashInstalledFont)
}

func parseFontInventory(raw string, hashFile func(string) (string, error)) ([]exportdomain.FontFileEvidence, error) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) > 2048 {
		return nil, fmt.Errorf("font face budget exceeded")
	}
	faces := make([]exportdomain.FontFileEvidence, 0, len(lines))
	hashes := map[string]string{}
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 5 || fields[0] == "" || fields[2] == "" || fields[3] == "" || fields[4] == "" {
			return nil, fmt.Errorf("incomplete font inventory")
		}
		hash, ok := hashes[fields[2]]
		if !ok {
			var err error
			hash, err = hashFile(fields[2])
			if err != nil {
				return nil, err
			}
			hashes[fields[2]] = hash
		}
		faces = append(faces, exportdomain.FontFileEvidence{PostScriptName: fields[0], Family: fields[1], Path: fields[2], FaceIndex: fields[3], FontVersion: fields[4], SHA256: hash, LicenseStatus: "not_reviewed"})
	}
	sort.Slice(faces, func(i, j int) bool {
		a, b := faces[i], faces[j]
		if a.PostScriptName != b.PostScriptName {
			return a.PostScriptName < b.PostScriptName
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.FaceIndex < b.FaceIndex
	})
	return faces, nil
}

func hashInstalledFont(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("font file unavailable")
	}
	// Only deployment font directories are eligible; Fontconfig must not cause
	// arbitrary user/config files to be read or disclosed in a review package.
	if !strings.HasPrefix(resolved, "/usr/share/fonts/") && !strings.HasPrefix(resolved, "/usr/local/share/fonts/") {
		return "", fmt.Errorf("font outside deployment font directories")
	}
	f, err := os.Open(resolved)
	if err != nil {
		return "", fmt.Errorf("font file unavailable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return "", fmt.Errorf("font file outside size/type budget")
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(f, (64<<20)+1))
	if err != nil || n != info.Size() {
		return "", fmt.Errorf("font changed while hashing")
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}
