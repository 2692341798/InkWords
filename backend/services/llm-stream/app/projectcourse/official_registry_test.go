package projectcourse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type countingOfficialFetcher struct{ calls int }

func (f *countingOfficialFetcher) Fetch(string) (string, error) {
	f.calls++
	return "official data that must remain data", nil
}

func TestOfficialRegistryProviderCachesByTechnologyVersionAndURL(t *testing.T) {
	fetcher := &countingOfficialFetcher{}
	provider := OfficialRegistryProvider{Registry: NewDefaultOfficialRegistry(), Fetcher: fetcher, Cache: NewMemoryOfficialSourceCache(), TTL: time.Hour, Now: func() time.Time { return time.Unix(1, 0) }}
	one, err := provider.FetchTechnology("Go", "1.25")
	require.NoError(t, err)
	two, err := provider.FetchTechnology("Go", "1.25")
	require.NoError(t, err)
	require.Equal(t, one.ContentHash, two.ContentHash)
	require.Equal(t, 1, fetcher.calls)
	require.Contains(t, OfficialSourcePromptBlock(one), "<official_source_data")
}
