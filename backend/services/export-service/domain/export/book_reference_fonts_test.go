package export

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBookReferenceUsesExplicitFontsWithoutThemeOverrides(t *testing.T) {
	archive, err := zip.OpenReader(filepath.Join("..", "..", "assets", "publishing", "inkwords-reference.docx"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, archive.Close()) })
	allowed := map[string]bool{"Noto Sans CJK SC": true, "Noto Sans Mono CJK SC": true, "FreeMono": true}
	declared, used := map[string]bool{}, map[string]bool{}
	for _, part := range archive.File {
		if !strings.HasPrefix(part.Name, "word/") || !strings.HasSuffix(part.Name, ".xml") {
			continue
		}
		reader, err := part.Open()
		require.NoError(t, err)
		decoder := xml.NewDecoder(reader)
		for {
			token, err := decoder.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			start, ok := token.(xml.StartElement)
			if !ok || start.Name.Space != "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				continue
			}
			for _, attribute := range start.Attr {
				if start.Name.Local == "rFonts" {
					require.NotContains(t, strings.ToLower(attribute.Name.Local), "theme", "%s overrides the intended font", part.Name)
					if attribute.Name.Local == "ascii" || attribute.Name.Local == "hAnsi" || attribute.Name.Local == "eastAsia" || attribute.Name.Local == "cs" {
						require.True(t, allowed[attribute.Value], "unexpected font %q in %s", attribute.Value, part.Name)
						used[attribute.Value] = true
					}
				}
				if part.Name == "word/fontTable.xml" && start.Name.Local == "font" && attribute.Name.Local == "name" {
					require.True(t, allowed[attribute.Value], "stale font table entry %q", attribute.Value)
					declared[attribute.Value] = true
				}
			}
		}
		require.NoError(t, reader.Close())
	}
	require.Equal(t, allowed, used)
	require.Equal(t, used, declared)
}

func TestBookReferenceKeepsComparisonRowsTogether(t *testing.T) {
	archive, err := zip.OpenReader(filepath.Join("..", "..", "assets", "publishing", "inkwords-reference.docx"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, archive.Close()) })
	for _, part := range archive.File {
		if part.Name != "word/styles.xml" {
			continue
		}
		reader, err := part.Open()
		require.NoError(t, err)
		var styles struct {
			Styles []struct {
				ID   string `xml:"styleId,attr"`
				Rows struct {
					CantSplit *struct{} `xml:"cantSplit"`
				} `xml:"trPr"`
			} `xml:"style"`
		}
		require.NoError(t, xml.NewDecoder(reader).Decode(&styles))
		require.NoError(t, reader.Close())
		for _, style := range styles.Styles {
			if style.ID == "Table" {
				require.NotNil(t, style.Rows.CantSplit, "comparison table rows may split across pages")
				return
			}
		}
	}
	t.Fatal("missing Table style")
}
