package main

import (
	"fmt"

	"github.com/google/uuid"
)

type deliveryAcknowledger interface {
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
}

func ackDelivery(delivery deliveryAcknowledger, reason string) error {
	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("ack %s: %w", reason, err)
	}
	return nil
}

func nackDelivery(delivery deliveryAcknowledger, taskID uuid.UUID) error {
	if err := delivery.Nack(false, true); err != nil {
		return fmt.Errorf("nack generation task %s: %w", taskID, err)
	}
	return nil
}
