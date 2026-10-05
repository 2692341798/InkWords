package bookrender

import (
	"strings"
	"testing"

	"github.com/chromedp/cdproto/css"
	"github.com/stretchr/testify/require"
)

func TestFontRulesRetainRenderingButNeverImportDirectoriesOrUserConfig(t *testing.T) {
	rules, err := extractFontRules([]byte(`<?xml version="1.0"?><fontconfig><dir>/unreviewed</dir><include>conf.d</include><cachedir>/cache</cachedir><match target="pattern"><edit name="family"><string>sans-serif</string></edit></match></fontconfig>`), true)
	require.NoError(t, err)
	require.Contains(t, string(rules), `<match target="pattern">`)
	require.NotContains(t, string(rules), "include")
	require.NotContains(t, string(rules), "/unreviewed")
	for _, xml := range []string{
		`<fontconfig><include>/user.conf</include></fontconfig>`,
		`<fontconfig><dir>/fonts</dir></fontconfig>`,
		`<fontconfig><match><edit name="file"><string>/outside.otf</string></edit></match></fontconfig>`,
		`<fontconfig><match><include>/nested</include></match></fontconfig>`,
		`<fontconfig><remap-dir as-path="/fonts">/outside</remap-dir></fontconfig>`,
	} {
		_, err := extractFontRules([]byte(xml), false)
		require.Error(t, err, xml)
	}
	_, err = extractFontRules([]byte(`<fontconfig><match>`), false)
	require.Error(t, err)
}

func TestObservedFontCoverageRejectsTofuEvenWhenAnotherInstalledFontHasTheGlyph(t *testing.T) {
	p := renderFontProfile{charsets: map[string]string{"Latin": "20-7e", "Chinese": "4e00-9fff"}}
	latinOnly := []*css.PlatformFontUsage{{PostScriptName: "Latin", GlyphCount: 2}}
	require.NoError(t, p.validateObservedCharacters(latinOnly, "ab"))
	require.Error(t, p.validateObservedCharacters(latinOnly, "中文"))
	require.NoError(t, p.validateObservedCharacters(append(latinOnly, &css.PlatformFontUsage{PostScriptName: "Chinese", GlyphCount: 2}), "ab中文"))
	require.Error(t, p.validateObservedCharacters([]*css.PlatformFontUsage{{PostScriptName: "Latin", IsCustomFont: true, GlyphCount: 1}}, "a"))
}

func TestFontProfileOverridesInheritedFontconfigEnvironment(t *testing.T) {
	env := fontProfileEnvironment([]string{"PATH=/bin", "FONTCONFIG_FILE=/untrusted", "FONTCONFIG_PATH=/untrusted", "FONTCONFIG_SYSROOT=/untrusted", "FC_DEBUG=999"}, "/render/fonts.conf")
	require.Contains(t, env, "PATH=/bin")
	require.Contains(t, env, "FONTCONFIG_FILE=/render/fonts.conf")
	for _, value := range env {
		require.NotContains(t, value, "/untrusted")
		require.False(t, strings.HasPrefix(value, "FC_DEBUG="))
		require.False(t, strings.HasPrefix(value, "FONTCONFIG_SYSROOT="))
	}
}

func TestFontProfileRejectsInheritedSysrootBeforeReadingDeploymentFiles(t *testing.T) {
	for _, value := range []string{"", "/", "/unreviewed"} {
		t.Setenv("FONTCONFIG_SYSROOT", value)
		_, err := prepareFontProfile(t.Context(), t.TempDir())
		require.ErrorContains(t, err, "FONTCONFIG_SYSROOT is incompatible")
	}
}

func TestFontProfileCharsetRejectsMissingOrMalformedGlyphs(t *testing.T) {
	require.NoError(t, requireFontCharacters("20-7e 5171 7b2c 9875\n", "第 12 页 / 共 16 页"))
	require.ErrorContains(t, requireFontCharacters("20-7e", "第 1 页"), "U+7B2C")
	require.Error(t, requireFontCharacters("20-xyz", "abc"))
	require.Error(t, requireFontCharacters("ffff-20", "abc"))
	require.NoError(t, requireFontCharacters("21a9", "\u21a9\ufe0e"))
	require.Error(t, requireFontCharacters("20-7e", "\u21a9\ufe0e"), "the base glyph is still required")
}
