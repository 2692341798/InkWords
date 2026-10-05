package bookrender

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFontInventoryPreservesCollectionFacesAndHashesEachFileOnce(t *testing.T) {
	calls := 0
	raw := "FaceB\tFamily B\t/usr/share/fonts/collection.ttc\t2\t131334\nFaceA\tFamily A\t/usr/share/fonts/collection.ttc\t1\t131334\n"
	faces, err := parseFontInventory(raw, func(path string) (string, error) {
		calls++
		require.Equal(t, "/usr/share/fonts/collection.ttc", path)
		return "sha256:" + strings.Repeat("a", 64), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, faces, 2)
	require.Equal(t, "FaceA", faces[0].PostScriptName)
	require.Equal(t, "1", faces[0].FaceIndex)
	require.Equal(t, "not_reviewed", faces[0].LicenseStatus)
}

func TestFontInventoryOutputLimitAlsoAppliesToIOCopy(t *testing.T) {
	var limited fontListOutput
	// Hide strings.Reader.WriteTo so io.Copy exercises the same destination
	// fast-path decision used when os/exec drains a pipe.
	source := struct{ io.Reader }{strings.NewReader(strings.Repeat("x", (1<<20)+1))}
	_, err := io.Copy(&limited, source)
	require.Error(t, err)
	require.LessOrEqual(t, limited.Len(), 1<<20)
}

func TestFontInventoryFailsClosedOnMissingMetadataAndUntrustedFiles(t *testing.T) {
	_, err := parseFontInventory("missing-fields", func(string) (string, error) { t.Fatal("must not read files for malformed inventory"); return "", nil })
	require.Error(t, err)
	_, err = parseFontInventory("Face\tFamily\t/outside/font.ttf\t0\t1", func(string) (string, error) { return "", fmt.Errorf("untrusted file") })
	require.Error(t, err)
	_, err = hashInstalledFont("/etc/hosts")
	require.ErrorContains(t, err, "outside deployment")
	var limited fontListOutput
	_, err = limited.Write(make([]byte, (1<<20)+1))
	require.Error(t, err)
}
