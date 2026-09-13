package services_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Keep browser protocol dependencies out of textbook domain/application code.
func TestBrowserSDKIsOwnedByExportInfrastructure(t *testing.T) {
	err := filepath.WalkDir("export-service", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, item := range file.Imports {
			value, _ := strconv.Unquote(item.Path.Value)
			if strings.HasPrefix(value, "github.com/chromedp/") {
				require.True(t, strings.HasPrefix(filepath.ToSlash(path), "export-service/infra/bookrender/"), path)
			}
		}
		return nil
	})
	require.NoError(t, err)
}
