package bootstrap

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodePlaywrightBrowserCaptureKeepsOnlyStructuredProbeOutput(t *testing.T) {
	capture, err := decodePlaywrightBrowserCapture(`{
  "url":"http://127.0.0.1:38080/route-registration",
  "final_url":"http://127.0.0.1:38080/route-registration",
  "browser_name":"Chromium",
  "browser_version":"151.0.7922.34",
  "playwright_version":"1.62.1",
  "screenshot_base64":"` + base64.StdEncoding.EncodeToString([]byte("png")) + `",
  "dom_assertions":[{"locator":"text=路由已注册","assertion":"has_text","expected":"路由已注册"}],
  "console":[],
  "network":[{"url":"http://127.0.0.1:38080/route-registration","resource_type":"document","status":200}]
}`)

	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:38080/route-registration", capture.URL)
	require.Equal(t, "Chromium", capture.BrowserName)
	require.Equal(t, "151.0.7922.34", capture.BrowserVersion)
	require.Equal(t, "1.62.1", capture.PlaywrightVersion)
	require.Equal(t, []byte("png"), capture.ScreenshotContent)
	require.Len(t, capture.DOMAssertions, 1)
	require.Len(t, capture.Network, 1)
}

func TestDecodePlaywrightBrowserCaptureFailsClosedForBadOrOversizedScreenshots(t *testing.T) {
	_, err := decodePlaywrightBrowserCapture(`{"screenshot_base64":"not base64"}`)
	require.ErrorContains(t, err, "screenshot")

	overBudget := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 10*1024*1024+1)))
	_, err = decodePlaywrightBrowserCapture(`{"screenshot_base64":"` + overBudget + `"}`)
	require.ErrorContains(t, err, "screenshot")

	_, err = decodePlaywrightBrowserCapture(`{"screenshot_base64":"` + base64.StdEncoding.EncodeToString([]byte("png")) + `","browser_name":"Chromium"}`)
	require.ErrorContains(t, err, "version")
}
