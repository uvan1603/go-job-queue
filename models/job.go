package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Job represents a background job in the queue
type Job struct {
	ID         primitive.ObjectID 								`bson:"_id,omitempty" json:"id"`
	Type       string             								`bson:"type" json:"type"`
	Payload    map[string]interface{}             `bson:"payload" json:"payload"`
	RetryCount int                							  `bson:"retryCount" json:"retryCount"`
	Status     string             								`bson:"status" json:"status"` // pending, processing, completed, failed
	CreatedAt  time.Time          								`bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          								`bson:"updatedAt" json:"updatedAt"`
}

// Valid statuses for a job
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	MaxRetries = 3
)
