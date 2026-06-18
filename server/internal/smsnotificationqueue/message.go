package smsnotificationqueue

import "github.com/google/uuid"

// QueueMessage is the RabbitMQ payload for one SMS notification delivery job.
type QueueMessage struct {
	UserID    uuid.UUID `json:"userId"`
	EventType string    `json:"eventType"`
	Phone     string    `json:"phone"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	ActionURL string    `json:"actionUrl,omitempty"`
}