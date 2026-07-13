package projectcourse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

type FileRole string

const (
	RoleEntrypoint     FileRole = "entrypoint"
	RoleDomain         FileRole = "domain"
	RoleApplication    FileRole = "application"
	RoleTransport      FileRole = "transport"
	RoleInfrastructure FileRole = "infrastructure"
	RoleConfiguration  FileRole = "configuration"
	RoleTest           FileRole = "test"
	RoleExample        FileRole = "example"
	RoleDocumentation  FileRole = "documentation"
	RoleBuildDeploy    FileRole = "build_deploy"
	RoleGenerated      FileRole = "generated"
	RoleBinary         FileRole = "binary"
	RoleUnknown        FileRole = "unknown"
)

type Disposition string

const (
	DispositionCovered  Disposition = "covered"
	DispositionIndexed  Disposition = "indexed"
	DispositionExcluded Disposition = "excluded"
)

type InventoryInput struct {
	Path    string
	Content []byte
	Mode    uint32
	IsTree  bool
}

type InventoryEntry struct {
	Path        string      `json:"path"`
	Role        FileRole    `json:"role"`
	Disposition Disposition `json:"disposition"`
	Reason      string      `json:"reason,omitempty"`
	ContentHash string      `json:"content_hash"`
	Size        int         `json:"size"`
	Content     string      `json:"content,omitempty"`
}

type InventoryOptions struct {
	MaxContentBytes int
}

func BuildInventory(inputs []InventoryInput, options InventoryOptions) ([]InventoryEntry, error) {
	entries := make([]InventoryEntry, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		cleanPath, err := normalizeInventoryPath(input.Path)
		if err != nil {
			return nil, err
		}
		if input.IsTree {
			continue
		}
		if _, ok := seen[cleanPath]; ok {
			return nil, fmt.Errorf("duplicate inventory path %q", cleanPath)
		}
		seen[cleanPath] = struct{}{}
		entry := classify(cleanPath, input.Content)
		entry.ContentHash = contentHash(input.Content)
		entry.Size = len(input.Content)
		if entry.Disposition != DispositionExcluded && options.MaxContentBytes > 0 && len(input.Content) <= options.MaxContentBytes {
			entry.Content = string(input.Content)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		pi, pj := rolePriority(entries[i].Role), rolePriority(entries[j].Role)
		if pi != pj {
			return pi < pj
		}
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].ContentHash < entries[j].ContentHash
	})
	return entries, nil
}

func normalizeInventoryPath(value string) (string, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if value == "" || strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("invalid repository path %q", value)
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("path traversal is not allowed: %q", value)
	}
	return clean, nil
}

func classify(filePath string, content []byte) InventoryEntry {
	role := RoleUnknown
	base := path.Base(filePath)
	lower := strings.ToLower(filePath)
	switch {
	case strings.Contains(lower, "/testdata/") || strings.HasSuffix(lower, "_test.go") || strings.HasSuffix(lower, ".test.ts") || strings.HasSuffix(lower, ".spec.ts") || strings.HasSuffix(lower, ".test.tsx"):
		role = RoleTest
	case strings.HasPrefix(lower, "docs/") || strings.HasSuffix(lower, "readme.md") || strings.HasSuffix(lower, ".md"):
		role = RoleDocumentation
	case strings.HasPrefix(lower, "examples/") || strings.HasPrefix(lower, "scripts/"):
		role = RoleExample
	case base == "dockerfile" || strings.HasPrefix(lower, ".github/") || strings.Contains(lower, "docker-compose") || strings.HasSuffix(lower, "nginx.conf"):
		role = RoleBuildDeploy
	case base == "go.mod" || base == "go.sum" || base == "package.json" || strings.HasSuffix(base, "lock") || strings.HasSuffix(base, ".yml") || strings.HasSuffix(base, ".yaml"):
		role = RoleConfiguration
	case strings.HasPrefix(lower, "backend/services/") && strings.Contains(lower, "/transport/") || strings.Contains(lower, "/handler") || strings.Contains(lower, "/route"):
		role = RoleTransport
	case strings.Contains(lower, "/infra/") || strings.Contains(lower, "consumer") || strings.Contains(lower, "client"):
		role = RoleInfrastructure
	case strings.Contains(lower, "/domain/") || strings.Contains(lower, "/model") || strings.Contains(lower, "/entity"):
		role = RoleDomain
	case strings.Contains(lower, "/app/") || strings.HasPrefix(lower, "cmd/") || strings.HasSuffix(lower, "main.go"):
		role = RoleApplication
	case strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".pdf") || strings.HasSuffix(lower, ".zip"):
		role = RoleBinary
	}
	if strings.Contains(lower, "/generated/") || strings.HasSuffix(lower, ".gen.go") {
		role = RoleGenerated
	}
	if role == RoleBinary {
		return InventoryEntry{Path: filePath, Role: role, Disposition: DispositionExcluded, Reason: "binary_or_archive"}
	}
	if role == RoleGenerated {
		return InventoryEntry{Path: filePath, Role: role, Disposition: DispositionIndexed, Reason: "generated_source"}
	}
	if !utf8.Valid(content) {
		return InventoryEntry{Path: filePath, Role: RoleBinary, Disposition: DispositionExcluded, Reason: "non_utf8"}
	}
	if role == RoleDocumentation || role == RoleExample || role == RoleTest || role == RoleConfiguration {
		return InventoryEntry{Path: filePath, Role: role, Disposition: DispositionIndexed}
	}
	return InventoryEntry{Path: filePath, Role: role, Disposition: DispositionCovered}
}

func contentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func rolePriority(role FileRole) int {
	for i, candidate := range []FileRole{RoleEntrypoint, RoleTransport, RoleApplication, RoleDomain, RoleInfrastructure, RoleConfiguration, RoleTest, RoleExample, RoleDocumentation, RoleBuildDeploy, RoleGenerated, RoleBinary, RoleUnknown} {
		if role == candidate {
			return i
		}
	}
	return 99
}
