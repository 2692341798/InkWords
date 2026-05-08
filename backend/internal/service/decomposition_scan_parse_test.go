package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"inkwords-backend/internal/parser"
)

func TestParseGithubOwnerRepo(t *testing.T) {
	tests := []struct {
		url           string
		expectedOwner string
		expectedRepo  string
		expectedOk    bool
	}{
		{"https://github.com/samber/lo", "samber", "lo", true},
		{"https://github.com/samber/lo.git", "samber", "lo", true},
		{"http://github.com/samber/lo/", "samber", "lo", true},
		{"git@github.com:samber/lo.git", "samber", "lo", true},
		{"https://gitlab.com/samber/lo", "", "", false},
		{"https://github.com/samber", "", "", false}, // invalid
	}

	for _, tt := range tests {
		owner, repo, ok := parser.ParseGithubOwnerRepo(tt.url)
		assert.Equal(t, tt.expectedOk, ok, "url: %s", tt.url)
		if ok {
			assert.Equal(t, tt.expectedOwner, owner, "url: %s", tt.url)
			assert.Equal(t, tt.expectedRepo, repo, "url: %s", tt.url)
		}
	}
}
