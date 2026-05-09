package parser

import (
	"path/filepath"
	"strings"
)

func isIgnoredPath(path string) bool {
	ignoredDirs := []string{
		"node_modules", "vendor", "dist", "build", "out", "target", "bin",
		".git", ".svn", ".idea", ".vscode", "__pycache__", "testdata", "docs", "examples", "scripts", "assets",
	}

	for _, dir := range ignoredDirs {
		if strings.Contains(path, "/"+dir+"/") || strings.HasPrefix(path, dir+"/") {
			return true
		}
	}

	name := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(name, "_test.go") ||
		strings.HasSuffix(name, ".test.js") ||
		strings.HasSuffix(name, ".spec.js") ||
		strings.HasSuffix(name, ".test.ts") ||
		strings.HasSuffix(name, ".spec.ts") {
		return true
	}

	ignoredExts := []string{
		".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".mp4", ".mp3", ".wav", ".zip", ".tar", ".gz", ".rar", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".ttf", ".woff", ".woff2", ".eot",
	}
	for _, ext := range ignoredExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func IsBinaryExt(ext string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".ico": true, ".webp": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".mp4": true, ".mp3": true, ".wav": true, ".avi": true, ".mov": true,
		".ttf": true, ".woff": true, ".woff2": true, ".eot": true,
		".pyc": true, ".class": true, ".jar": true, ".war": true,
	}
	return binaryExts[ext]
}
