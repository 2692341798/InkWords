package stream

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestExternalStreamErrorMessage_HidesInternalGenerateErrors(t *testing.T) {
	require.Equal(t, "blog generation failed", externalStreamErrorMessage(streamOperationGenerate, errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")))
}

func TestExternalStreamErrorMessage_MapsContinueNotFoundToStableMessage(t *testing.T) {
	err := errors.New("blog not found: " + gorm.ErrRecordNotFound.Error())
	require.Equal(t, "blog not found", externalStreamErrorMessage(streamOperationContinue, err))
}

func TestExternalStreamErrorMessage_HidesInternalAnalyzeErrors(t *testing.T) {
	require.Equal(t, "blog analysis failed", externalStreamErrorMessage(streamOperationAnalyze, errors.New("failed to create temp dir")))
}
