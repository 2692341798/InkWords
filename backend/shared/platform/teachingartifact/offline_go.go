package teachingartifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/version"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// Dependency bundles have their own budget; ordinary teaching trees stay at 8 MB.
const (
	GoDependencyManifestFormat       = "inkwords.go-offline-dependencies.v1"
	MaxGoDependencyBytes       int64 = 64 << 20
	maxGoDependencyFiles             = 4096
)

var exactDependencyToolchain = regexp.MustCompile(`^go[0-9]+\.[0-9]+\.[0-9]+$`)

// GoDependencyOrigin binds an operator-selected module to its source revision.
// The lock records this provenance; hashing alone does not prove its truth.
type GoDependencyOrigin struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// GoDependencyModule pins both Go module checksum records.
type GoDependencyModule struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Sum      string `json:"sum"`
	GoModSum string `json:"go_mod_sum"`
}

// GoDependencyFile binds a relative file path, length and content digest.
type GoDependencyFile struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// GoDependencyManifest is an operator-owned inventory, never a manuscript or
// learner request. Its expected hash must come from reviewed configuration.
type GoDependencyManifest struct {
	Format    string               `json:"format"`
	Origin    GoDependencyOrigin   `json:"origin"`
	Toolchain string               `json:"toolchain"`
	Modules   []GoDependencyModule `json:"modules"`
	Files     []GoDependencyFile   `json:"files"`
}

// OfflineGoBundle contains detached verified bytes. It neither grants runtime
// approval nor asserts that the dependencies' publication rights are cleared.
type OfflineGoBundle struct {
	hash     string
	manifest GoDependencyManifest
	files    []File
}

// Hash returns the operator-selected inventory digest.
func (b *OfflineGoBundle) Hash() string { return b.hash }

// Origin returns the pinned primary module provenance.
func (b *OfflineGoBundle) Origin() GoDependencyOrigin { return b.manifest.Origin }

// Toolchain returns the exact runtime version named in go.mod and the inventory.
func (b *OfflineGoBundle) Toolchain() string { return b.manifest.Toolchain }

// Files returns owned copies so later caller mutations cannot alter the bundle.
func (b *OfflineGoBundle) Files() []File {
	result := make([]File, len(b.files))
	for i, file := range b.files {
		result[i] = File{Path: file.Path, Content: bytes.Clone(file.Content)}
	}
	return result
}

// LoadOfflineGoBundle reads only a bounded regular-file vendor tree whose exact
// inventory hash was selected by the operator. It never invokes Go or downloads.
func LoadOfflineGoBundle(ctx context.Context, root string, manifestJSON []byte, expectedHash string) (*OfflineGoBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(manifestJSON) == 0 || len(manifestJSON) > 2<<20 || dependencyDigest(manifestJSON) != expectedHash {
		return nil, fmt.Errorf("offline dependency manifest hash mismatch")
	}
	var manifest GoDependencyManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode dependency inventory: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("trailing dependency inventory data")
	}
	index, err := dependencyFileIndex(manifest)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("invalid operator dependency directory")
	}
	files := make([]File, 0, len(index)+1)
	err = filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if name == root {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			for key := range index {
				if strings.HasPrefix(key, relative+"/") {
					return nil
				}
			}
			return fmt.Errorf("unexpected dependency directory")
		}
		pinned, ok := index[relative]
		if !ok || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected or unsafe dependency file")
		}
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		content, readErr := io.ReadAll(io.LimitReader(file, pinned.Bytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if int64(len(content)) != pinned.Bytes || dependencyDigest(content) != pinned.SHA256 {
			return fmt.Errorf("offline dependency file hash mismatch: %s", relative)
		}
		files = append(files, File{Path: relative, Content: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) != len(index) {
		return nil, fmt.Errorf("offline dependency files are missing")
	}
	if err := validateDependencyMetadata(manifest, files); err != nil {
		return nil, err
	}
	files = append(files, File{Path: "inkwords-dependencies.json", Content: bytes.Clone(manifestJSON)})
	return &OfflineGoBundle{hash: expectedHash, manifest: manifest, files: files}, nil
}

func dependencyFileIndex(manifest GoDependencyManifest) (map[string]GoDependencyFile, error) {
	commit, err := hex.DecodeString(manifest.Origin.Commit)
	if manifest.Format != GoDependencyManifestFormat || err != nil || len(commit) != 20 || manifest.Origin.Commit != strings.ToLower(manifest.Origin.Commit) || module.Check(manifest.Origin.Module, manifest.Origin.Version) != nil || !version.IsValid(manifest.Toolchain) || !exactDependencyToolchain.MatchString(manifest.Toolchain) || version.Compare(manifest.Toolchain, "go1.26.0") < 0 || len(manifest.Files) < 3 || len(manifest.Files) > maxGoDependencyFiles || len(manifest.Modules) == 0 || len(manifest.Modules) > 256 {
		return nil, fmt.Errorf("invalid offline dependency inventory")
	}
	index := make(map[string]GoDependencyFile, len(manifest.Files))
	var total int64
	for _, file := range manifest.Files {
		_, duplicate := index[file.Path]
		digest, err := tokenDigest(file.SHA256)
		if duplicate || file.Path != path.Clean(file.Path) || strings.ContainsAny(file.Path, "\\\x00\r\n") || (file.Path != "go.mod" && file.Path != "go.sum" && !strings.HasPrefix(file.Path, "vendor/")) || file.Bytes < 0 || file.Bytes > MaxGoDependencyBytes-total || err != nil || file.SHA256 != "sha256:"+digest {
			return nil, fmt.Errorf("invalid offline dependency file inventory")
		}
		index[file.Path] = file
		total += file.Bytes
	}
	return index, nil
}

func validateDependencyMetadata(manifest GoDependencyManifest, files []File) error {
	content := make(map[string][]byte, len(files))
	for _, file := range files {
		content[file.Path] = file.Content
	}
	parsed, err := modfile.Parse("go.mod", content["go.mod"], nil)
	if err != nil || parsed.Module == nil || parsed.Module.Mod.Path != "example.com/inkwords/teaching" || parsed.Go == nil || parsed.Go.Version != "1.26.0" || parsed.Toolchain == nil || parsed.Toolchain.Name != manifest.Toolchain {
		return fmt.Errorf("offline dependency Go module metadata mismatch")
	}
	for _, statement := range parsed.Syntax.Stmt {
		var tokens []string
		switch line := statement.(type) {
		case *modfile.Line:
			tokens = line.Token
		case *modfile.LineBlock:
			tokens = line.Token
		}
		if len(tokens) == 0 {
			continue
		}
		switch tokens[0] {
		case "module", "go", "toolchain", "require":
		default:
			return fmt.Errorf("unsupported offline dependency module directive")
		}
	}
	pins := make(map[string]GoDependencyModule, len(manifest.Modules))
	for _, pin := range manifest.Modules {
		_, duplicate := pins[pin.Path]
		if duplicate || module.Check(pin.Path, pin.Version) != nil || !dependencyModuleSum(pin.Sum) || !dependencyModuleSum(pin.GoModSum) {
			return fmt.Errorf("invalid offline module pin")
		}
		pins[pin.Path] = pin
	}
	if pins[manifest.Origin.Module].Version != manifest.Origin.Version || len(pins) != len(parsed.Require) {
		return fmt.Errorf("offline module origin or require list mismatch")
	}
	for _, required := range parsed.Require {
		if pins[required.Mod.Path].Version != required.Mod.Version {
			return fmt.Errorf("offline required version mismatch")
		}
	}
	sums := make(map[string]string)
	for _, line := range strings.Split(string(content["go.sum"]), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || !dependencyModuleSum(fields[2]) {
			return fmt.Errorf("invalid offline module checksum record")
		}
		key := fields[0] + " " + fields[1]
		if previous, ok := sums[key]; ok && previous != fields[2] {
			return fmt.Errorf("conflicting offline module checksums")
		}
		sums[key] = fields[2]
	}
	for _, pin := range pins {
		if sums[pin.Path+" "+pin.Version] != pin.Sum || sums[pin.Path+" "+pin.Version+"/go.mod"] != pin.GoModSum {
			return fmt.Errorf("offline module checksum binding mismatch")
		}
	}
	vendorModules := make(map[string]string)
	for _, line := range strings.Split(string(content["vendor/modules.txt"]), "\n") {
		if !strings.HasPrefix(line, "# ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || pins[fields[1]].Version != fields[2] {
			return fmt.Errorf("offline vendor version mismatch")
		}
		if _, exists := vendorModules[fields[1]]; exists {
			return fmt.Errorf("duplicate offline vendor module")
		}
		vendorModules[fields[1]] = fields[2]
	}
	if len(vendorModules) != len(pins) {
		return fmt.Errorf("offline vendor modules are missing")
	}
	primaryFiles := 0
	for name := range content {
		if !strings.HasPrefix(name, "vendor/") || name == "vendor/modules.txt" {
			continue
		}
		owned := false
		for modulePath := range pins {
			if strings.HasPrefix(name, "vendor/"+modulePath+"/") {
				owned = true
				break
			}
		}
		if !owned {
			return fmt.Errorf("offline vendor file has no pinned module")
		}
		if strings.HasPrefix(name, "vendor/"+manifest.Origin.Module+"/") {
			primaryFiles++
		}
	}
	if primaryFiles == 0 {
		return fmt.Errorf("primary offline module has no files")
	}
	return nil
}

func dependencyModuleSum(value string) bool {
	if !strings.HasPrefix(value, "h1:") {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "h1:"))
	return err == nil && len(decoded) == sha256.Size
}

func dependencyDigest(content []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(content)) }
