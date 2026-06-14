package obsidian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
)

func TestRestAPIStoreReadSetsAuthHeader(t *testing.T) {
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

	store := NewRestAPIStore(u, "dummy", srv.Client())
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

func TestRestAPIStoreListParsesStringArray(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["a.md","b.md"]`))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	store := NewRestAPIStore(u, "dummy", srv.Client())

	items, err := store.List(context.Background(), "wiki/concepts")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 || items[0] != "a.md" || items[1] != "b.md" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestRestAPIStoreListParsesFilesObject(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"files":["a.md","b.md","concepts/"]}`))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	store := NewRestAPIStore(u, "dummy", srv.Client())

	items, err := store.List(context.Background(), "wiki")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 || items[0] != "a.md" || items[1] != "b.md" || items[2] != "concepts/" {
		t.Fatalf("unexpected items: %#v", items)
	}
}
