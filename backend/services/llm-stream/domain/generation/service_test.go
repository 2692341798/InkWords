package generation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewService_UsesServiceOwnedDependencies(t *testing.T) {
	svc := NewService(nil, nil)
	require.NotNil(t, svc)
	require.Nil(t, svc.blogWriter)
	require.Nil(t, svc.taskWriter)
}
