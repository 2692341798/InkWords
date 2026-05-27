package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReviewSessionBeforeCreate_AssignsUUID(t *testing.T) {
	session := ReviewSession{}

	err := session.BeforeCreate(&gorm.DB{})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, session.ID)
}

func TestReviewTurnBeforeCreate_AssignsUUID(t *testing.T) {
	turn := ReviewTurn{}

	err := turn.BeforeCreate(&gorm.DB{})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, turn.ID)
}
