package stream

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type testStreamWriter struct {
	*httptest.ResponseRecorder
	flushCount int
}

func newTestStreamWriter() *testStreamWriter {
	return &testStreamWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (w *testStreamWriter) CloseNotify() <-chan bool {
	ch := make(chan bool)
	return ch
}

func (w *testStreamWriter) Flush() {
	w.flushCount++
}

func (w *testStreamWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func (w *testStreamWriter) Pusher() http.Pusher {
	return nil
}

func (w *testStreamWriter) Status() int {
	return w.Code
}

func (w *testStreamWriter) Size() int {
	return w.Body.Len()
}

func (w *testStreamWriter) Written() bool {
	return w.Code != 0 || w.Body.Len() > 0
}

func (w *testStreamWriter) WriteHeaderNow() {}

func (w *testStreamWriter) WriteString(s string) (int, error) {
	return w.Body.WriteString(s)
}

func TestNewGenerateStreamChannels_UsesBufferedChannels(t *testing.T) {
	chunkChan, errChan := newGenerateStreamChannels()

	require.Greater(t, cap(chunkChan), 0)
	require.Greater(t, cap(errChan), 0)
}

func TestWriteStreamEvent_FlushesAfterWriting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	writer := newTestStreamWriter()
	ctx, _ := gin.CreateTestContext(writer)

	writeStreamEvent(ctx, writer, "chunk", `{"status":"progress"}`)

	require.Equal(t, 1, writer.flushCount)
	require.Contains(t, writer.Body.String(), "event:chunk")
	require.Contains(t, writer.Body.String(), `data:{"status":"progress"}`)
}

type stubStreamService struct {
	checkQuotaFunc       func(uuid.UUID) error
	generateFunc         func(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error)
	continueFunc         func(context.Context, uuid.UUID, uuid.UUID, chan<- string, chan<- error)
	polishFunc           func(context.Context, PolishRequest, chan<- string, chan<- error)
	analyzeStreamFunc    func(context.Context, uuid.UUID, GenerateRequest, chan<- string, chan<- error)
	scanProjectModulesFn func(context.Context, string, chan<- string) ([]ModuleCard, error)
}

func (s *stubStreamService) CheckQuota(uid uuid.UUID) error {
	if s.checkQuotaFunc != nil {
		return s.checkQuotaFunc(uid)
	}
	return nil
}

func (s *stubStreamService) Generate(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
	if s.generateFunc != nil {
		s.generateFunc(ctx, userID, req, chunkChan, errChan)
	}
}

func (s *stubStreamService) Continue(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	if s.continueFunc != nil {
		s.continueFunc(ctx, userID, blogID, chunkChan, errChan)
	}
}

func (s *stubStreamService) Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
	if s.polishFunc != nil {
		s.polishFunc(ctx, req, chunkChan, errChan)
	}
}

func (s *stubStreamService) AnalyzeStream(ctx context.Context, userID uuid.UUID, req GenerateRequest, progressChan chan<- string, errChan chan<- error) {
	if s.analyzeStreamFunc != nil {
		s.analyzeStreamFunc(ctx, userID, req, progressChan, errChan)
	}
}

func (s *stubStreamService) ScanProjectModules(ctx context.Context, gitURL string, progressChan chan<- string) ([]ModuleCard, error) {
	if s.scanProjectModulesFn != nil {
		return s.scanProjectModulesFn(ctx, gitURL, progressChan)
	}
	return nil, nil
}

type stubBlogReadable struct {
	existsErr error
}

func (s stubBlogReadable) Exists(context.Context, uuid.UUID, uuid.UUID) error {
	return s.existsErr
}

func newJSONStreamTestContext(body string) (*gin.Context, *testStreamWriter, context.CancelFunc) {
	writer := newTestStreamWriter()
	ctx, _ := gin.CreateTestContext(writer)

	requestContext, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)).WithContext(requestContext)
	request.Header.Set("Content-Type", "application/json")
	ctx.Request = request

	return ctx, writer, cancel
}

func waitForContextCancellation(t *testing.T, cancel context.CancelFunc, started <-chan struct{}, observed <-chan error, done <-chan struct{}) {
	t.Helper()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("service did not start")
	}

	cancel()

	select {
	case err := <-observed:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("service did not observe request cancellation")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not stop after request cancellation")
	}
}

func TestContinueAnalyzeAndScanHandlers_PropagateRequestCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("continue uses request context", func(t *testing.T) {
		started := make(chan struct{}, 1)
		observed := make(chan error, 1)

		handler := &Handler{
			service: &stubStreamService{
				continueFunc: func(ctx context.Context, _ uuid.UUID, _ uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
					started <- struct{}{}
					<-ctx.Done()
					observed <- ctx.Err()
					close(chunkChan)
					close(errChan)
				},
			},
			blogRepo: stubBlogReadable{},
		}

		ctx, _, cancel := newJSONStreamTestContext("")
		ctx.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
		ctx.Set("user_id", uuid.New())

		done := make(chan struct{})
		go func() {
			handler.ContinueBlogStreamHandler(ctx)
			close(done)
		}()

		waitForContextCancellation(t, cancel, started, observed, done)
	})

	t.Run("analyze uses request context", func(t *testing.T) {
		started := make(chan struct{}, 1)
		observed := make(chan error, 1)

		handler := &Handler{
			service: &stubStreamService{
				analyzeStreamFunc: func(ctx context.Context, _ uuid.UUID, _ GenerateRequest, progressChan chan<- string, errChan chan<- error) {
					started <- struct{}{}
					<-ctx.Done()
					observed <- ctx.Err()
					close(progressChan)
					close(errChan)
				},
			},
			blogRepo: stubBlogReadable{},
		}

		ctx, _, cancel := newJSONStreamTestContext(`{"source_content":"parsed text"}`)

		done := make(chan struct{})
		go func() {
			handler.AnalyzeStreamHandler(ctx)
			close(done)
		}()

		waitForContextCancellation(t, cancel, started, observed, done)
	})

	t.Run("scan uses request context", func(t *testing.T) {
		started := make(chan struct{}, 1)
		observed := make(chan error, 1)

		handler := &Handler{
			service: &stubStreamService{
				scanProjectModulesFn: func(ctx context.Context, _ string, _ chan<- string) ([]ModuleCard, error) {
					started <- struct{}{}
					<-ctx.Done()
					observed <- ctx.Err()
					return nil, ctx.Err()
				},
			},
			blogRepo: stubBlogReadable{},
		}

		ctx, _, cancel := newJSONStreamTestContext(`{"git_url":"https://github.com/example/repo"}`)

		done := make(chan struct{})
		go func() {
			handler.ScanStreamHandler(ctx)
			close(done)
		}()

		waitForContextCancellation(t, cancel, started, observed, done)
	})
}
