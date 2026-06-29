package main

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeAcknowledger struct {
	ackErr  error
	nackErr error
	acked   int
	nacked  int
}

func (f *fakeAcknowledger) Ack(bool) error {
	f.acked++
	return f.ackErr
}

func (f *fakeAcknowledger) Nack(bool, bool) error {
	f.nacked++
	return f.nackErr
}

func TestAckDelivery_ReturnsWrappedFailure(t *testing.T) {
	ack := &fakeAcknowledger{ackErr: errors.New("channel closed")}
	err := ackDelivery(ack, "malformed generation message")
	require.ErrorContains(t, err, "malformed generation message")
	require.ErrorContains(t, err, "channel closed")
	require.Equal(t, 1, ack.acked)
}

func TestNackDelivery_ReturnsWrappedFailure(t *testing.T) {
	ack := &fakeAcknowledger{nackErr: errors.New("channel closed")}
	err := nackDelivery(ack, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	require.ErrorContains(t, err, "nack generation task")
	require.Equal(t, 1, ack.nacked)
}
