package service

import "context"

type ObsidianStore interface {
	Read(ctx context.Context, path string) ([]byte, error)
	Put(ctx context.Context, path string, contentType string, body []byte) error
	Post(ctx context.Context, path string, contentType string, body []byte) error
	Patch(ctx context.Context, path string, headers map[string]string, contentType string, body []byte) error
	List(ctx context.Context, dirPath string) ([]string, error)
}

