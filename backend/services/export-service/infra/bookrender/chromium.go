package bookrender

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	exportdomain "inkwords-backend/services/export-service/domain/export"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const defaultTimeout = 45 * time.Second
const layoutVersion = "inkwords.pdf-layout.v3"

// Chromium renders only the canonical HTML projection using a bounded browser
// instance. A service instance runs at most one PDF browser at a time.
type Chromium struct {
	executable string
	timeout    time.Duration
	slots      chan struct{}
	images     exportdomain.BookImageSource
}

// WithBookImages supplies only the frozen, content-addressed visual reader.
func (r *Chromium) WithBookImages(source exportdomain.BookImageSource) *Chromium {
	r.images = source
	return r
}

// NewChromium uses the trusted deployment executable and normal browser sandbox.
func NewChromium(executable string) *Chromium {
	return NewChromiumWithTimeout(executable, defaultTimeout)
}

// NewChromiumWithTimeout also bounds queueing and startup, not just printing.
func NewChromiumWithTimeout(executable string, timeout time.Duration) *Chromium {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Chromium{executable: strings.TrimSpace(executable), timeout: timeout, slots: make(chan struct{}, 1)}
}

// RenderPDF adds only browser-calculated page numbers to the frozen projection.
func (r *Chromium) RenderPDF(ctx context.Context, book sharedtextbook.CanonicalBookAST) (result exportdomain.BookPDFProjection, err error) {
	if r == nil || r.executable == "" {
		return result, fmt.Errorf("Chromium PDF renderer is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	case <-ctx.Done():
		return result, ctx.Err()
	}
	document, err := exportdomain.RenderBookHTML(book, r.images)
	if err != nil {
		return result, err
	}
	directory, err := os.MkdirTemp("", "inkwords-book-pdf-*")
	if err != nil {
		return result, fmt.Errorf("create PDF directory: %w", err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "book.html")
	if err := os.WriteFile(path, document, 0o600); err != nil {
		return result, fmt.Errorf("write canonical HTML: %w", err)
	}
	var log boundedLog
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		result.RenderLog = "status=" + status + "\nlayout=" + layoutVersion + "\nsandbox=required\noutput=" + log.String()
	}()
	options := browserOptions(r.executable, filepath.Join(directory, "profile"), &log)
	var profile *renderFontProfile
	mode := os.Getenv("TEXTBOOK_PDF_FONT_PROFILE")
	if mode != "" {
		if mode != reviewedFontProfile {
			return result, fmt.Errorf("unsupported PDF font profile")
		}
		profile, err = prepareFontProfile(ctx, directory)
		if err != nil {
			return result, fmt.Errorf("prepare PDF font source set: %w", err)
		}
		// chromedp merges with the host environment; neutralize inherited debug
		// there. fc-list uses the clean environment directly: even FC_DEBUG=0
		// prints a diagnostic prefix that is not part of its machine output.
		options = append(options, chromedp.Env(append(slices.Clone(profile.env), "FC_DEBUG=0")...))
	}
	allocator, stop := chromedp.NewExecAllocator(ctx, options...)
	defer stop()
	tab, closeTab := chromedp.NewContext(allocator, chromedp.WithLogf(func(format string, args ...any) { fmt.Fprintf(&log, format+"\n", args...) }))
	defer closeTab()
	var content []byte
	var version string
	var inventory []exportdomain.FontFileEvidence
	var inventoryErr error
	if profile != nil {
		inventory, inventoryErr = profile.inventory(ctx)
		if inventoryErr != nil {
			return result, inventoryErr
		}
	} else {
		inventory, inventoryErr = installedFontFaces(ctx)
	}
	var fontObservation bodyFontObservation
	var validateCharacters func([]*css.PlatformFontUsage, string) error
	if profile != nil {
		validateCharacters = profile.validateObservedCharacters
	}
	err = chromedp.Run(tab,
		chromedp.Navigate((&url.URL{Scheme: "file", Path: path}).String()),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
		chromedp.Evaluate(`Promise.all(Array.from(document.images, image => image.decode())).then(() => true)`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, product, _, _, _, versionErr := browser.GetVersion().Do(ctx)
			if versionErr != nil {
				return versionErr
			}
			version = product
			fontObservation = observeBodyFonts(ctx, inventory, validateCharacters)
			if profile != nil && fontObservation.err != nil {
				return fmt.Errorf("PDF body font coverage: %w", fontObservation.err)
			}
			var printErr error
			content, _, printErr = page.PrintToPDF().WithPrintBackground(true).WithPreferCSSPageSize(true).
				WithDisplayHeaderFooter(true).WithHeaderTemplate("<div></div>").WithFooterTemplate(bookFooter).
				WithGenerateTaggedPDF(true).WithGenerateDocumentOutline(true).Do(ctx)
			return printErr
		}))
	if err != nil {
		return result, fmt.Errorf("render canonical book PDF: %w", err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		return result, fmt.Errorf("Chromium output is not PDF")
	}
	if profile != nil {
		after, checkErr := profile.inventory(ctx)
		if checkErr != nil || !slices.Equal(inventory, after) {
			return result, fmt.Errorf("PDF font source set changed during printing")
		}
		profile.evidence.InventoryStable = true
		profile.evidence.BodyCharsetStatus = "covered_by_observed_font_candidates"
	}
	result.Content = content
	evidence := exportdomain.NewPDFFontEvidence(book, document, content, time.Now().UTC())
	evidence.RendererVersion = version
	if profile != nil {
		evidence.SourceSet = &profile.evidence
		evidence.Limitations = append(evidence.Limitations, "Fontconfig 字体选择被限定到本次已核对的部署集合；字符覆盖检查不等同于逐字形正确性或文件打开观测，也不覆盖图像内嵌字体或其它导出格式。")
		evidence.Limitations = append(evidence.Limitations, "字符覆盖检查不为变体选择符要求独立字形；基础字符仍必须覆盖，变体选择与复杂文字塑形效果仍需实际校样审阅。")
	}
	if inventoryErr == nil {
		evidence.InstalledFaces = inventory
	} else {
		evidence.Limitations = append(evidence.Limitations, "本次未能取得完整的部署字体文件清单。")
	}
	if len(fontObservation.fonts) > 0 {
		evidence.Fonts = fontObservation.fonts
		evidence.ObservedTextNodes = fontObservation.nodes
		evidence.BodyObservationStatus = "partial"
	}
	if fontObservation.err != nil {
		evidence.Limitations = append(evidence.Limitations, "正文节点观测中断或超出预算；已有观测不能代表整份文档。")
	}
	result.FontEvidence = &evidence
	result.MediaType = "application/pdf"
	result.ToolVersions = map[string]string{"chromium": version, "pdf_layout": layoutVersion, "chromedp": "v0.15.1", "book_images": "inkwords.frozen-book-images.v1"}
	return result, nil
}

const bookFooter = `<div style="width:100%;text-align:center;font-family:'Noto Sans CJK SC',sans-serif;font-size:12px;color:#606975">第 <span class="pageNumber"></span> 页 / 共 <span class="totalPages"></span> 页</div>`

func browserOptions(executable, profile string, log *boundedLog) []chromedp.ExecAllocatorOption {
	// Do not copy DefaultExecAllocatorOptions: some defaults disable browser
	// protections. Explicit false also suppresses chromedp's root auto-fallback.
	return []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(executable), chromedp.UserDataDir(profile), chromedp.Headless, chromedp.DisableGPU,
		chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("no-sandbox", false), chromedp.Flag("disable-setuid-sandbox", false),
		chromedp.Flag("disable-background-networking", true), chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-extensions", true), chromedp.CombinedOutput(log),
	}
}

type boundedLog struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *boundedLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := min(len(p), 4096-b.buffer.Len())
	if n > 0 {
		b.buffer.Write(p[:n])
	}
	return len(p), nil
}
func (b *boundedLog) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.buffer.String() }
