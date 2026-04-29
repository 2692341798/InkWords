package service

import (
	"bytes"
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
)

func TestNewRestAPIStoreFromEnv_CertMissing(t *testing.T) {
	t.Setenv("OBSIDIAN_REST_API_BASE_URL", "https://obsidian-bridge:27125")
	t.Setenv("OBSIDIAN_REST_API_KEY", "dummy")
	t.Setenv("OBSIDIAN_REST_API_CERT_PATH", "/path/not-exist.crt")

	_, err := newRestAPIStoreFromEnv()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNewRestAPIStoreFromEnv_AllowsInsecureSkipVerifyWithoutCert(t *testing.T) {
	t.Setenv("OBSIDIAN_REST_API_BASE_URL", "https://obsidian-bridge:27125")
	t.Setenv("OBSIDIAN_REST_API_KEY", "dummy")
	t.Setenv("OBSIDIAN_REST_API_CERT_PATH", "")
	t.Setenv("OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY", "true")

	_, err := newRestAPIStoreFromEnv()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRestAPIStore_Read_SetsAuthHeader(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	store := newRestAPIStore(u, "dummy", srv.Client())
	_, err = store.Read(context.Background(), "wiki/index.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if gotAuth != "Bearer dummy" {
		t.Fatalf("unexpected Authorization header: %q", gotAuth)
	}

	wantPath := path.Join("/vault", "wiki/index.md")
	if gotPath != wantPath {
		t.Fatalf("unexpected path: got %q want %q", gotPath, wantPath)
	}
}

func TestRestAPIStore_List_ParsesStringArray(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["a.md","b.md"]`))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	store := newRestAPIStore(u, "dummy", srv.Client())

	items, err := store.List(context.Background(), "wiki/concepts")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 || items[0] != "a.md" || items[1] != "b.md" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestRestAPIStoreFromEnv_LoadsCertFile(t *testing.T) {
	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tlsSrv.Close()

	cert := tlsSrv.Certificate()
	if cert == nil {
		t.Fatalf("expected certificate")
	}

	var pemBuf bytes.Buffer
	if err := writePEMCertificate(&pemBuf, cert.Raw); err != nil {
		t.Fatalf("pem: %v", err)
	}

	tmp, err := os.CreateTemp(t.TempDir(), "obsidian-cert-*.crt")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := tmp.Write(pemBuf.Bytes()); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = tmp.Close()

	t.Setenv("OBSIDIAN_REST_API_BASE_URL", tlsSrv.URL)
	t.Setenv("OBSIDIAN_REST_API_KEY", "dummy")
	t.Setenv("OBSIDIAN_REST_API_CERT_PATH", tmp.Name())

	_, err = newRestAPIStoreFromEnv()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func writePEMCertificate(buf *bytes.Buffer, der []byte) error {
	return pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: der})
}
