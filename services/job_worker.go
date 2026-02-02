package services

import (
	"context"
	"log"
	"time"

	"jobqueue/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// JobWorker processes jobs from the queue
type JobWorker struct {
	jobQueue   chan primitive.ObjectID
	jobsCol    *mongo.Collection
	stopChan   chan struct{}
	numWorkers int
}

// NewJobWorker creates a new job worker with the specified number of worker goroutines
func NewJobWorker(jobsCollection *mongo.Collection, queueSize int, numWorkers int) *JobWorker {
	return &JobWorker{
		jobQueue:   make(chan primitive.ObjectID, queueSize),
		jobsCol:    jobsCollection,
		stopChan:   make(chan struct{}),
		numWorkers: numWorkers,
	}
}

// Start initializes and starts worker goroutines to process jobs
// Each worker reads job IDs from the job queue channel and processes them
func (jw *JobWorker) Start() {
	log.Printf("Starting %d job worker(s)\n", jw.numWorkers)

	// Start multiple worker goroutines
	for i := 1; i <= jw.numWorkers; i++ {
		go jw.worker(i)
	}
}

// worker is a single worker goroutine that processes jobs
// It continuously reads job IDs from the jobQueue channel and updates their status
func (jw *JobWorker) worker(id int) {
	log.Printf("Worker %d started\n", id)

	for {
		select {
		case jobID := <-jw.jobQueue:
			// Process the job
			jw.processJob(jobID)

		case <-jw.stopChan:
			log.Printf("Worker %d stopped\n", id)
			return
		}
	}
}

// processJob updates the job status through its lifecycle
func (jw *JobWorker) processJob(jobID primitive.ObjectID) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch the job
	var job models.Job
	err := jw.jobsCol.FindOne(ctx, bson.M{"_id": jobID}).Decode(&job)
	if err != nil {
		log.Printf("Failed to find job %s: %v\n", jobID.Hex(), err)
		return
	}

	// Update status to processing
	_, err = jw.jobsCol.UpdateOne(ctx, bson.M{"_id": jobID}, bson.M{
		"$set": bson.M{"status": models.StatusProcessing, "updatedAt": time.Now()},
	})
	if err != nil {
		log.Printf("Failed to update job %s to processing: %v\n", jobID.Hex(), err)
		return
	}

	log.Printf("Processing job: %s\n", jobID.Hex())

	// Simulate execution: fail if payload has "fail": true
	if val, ok := job.Payload["fail"].(bool); ok && val {
		// Increment retry count
		newRetryCount := job.RetryCount + 1
		update := bson.M{
			"status":     models.StatusFailed,
			"retryCount": newRetryCount,
			"updatedAt":  time.Now(),
		}
		_, err = jw.jobsCol.UpdateOne(ctx, bson.M{"_id": jobID}, bson.M{"$set": update})
		if err != nil {
			log.Printf("Failed to mark job %s as failed: %v\n", jobID.Hex(), err)
		}

		log.Printf("Job %s failed (retry count: %d)\n", jobID.Hex(), newRetryCount)

		// Requeue if retries < MaxRetries
		const MaxRetries = 3
		if newRetryCount < MaxRetries {
			log.Printf("Re-enqueueing job %s for retry\n", jobID.Hex())
			jw.EnqueueJob(jobID)
		}
		return
	}

	// Normal successful execution
	time.Sleep(2 * time.Second)

	_, err = jw.jobsCol.UpdateOne(ctx, bson.M{"_id": jobID}, bson.M{
		"$set": bson.M{"status": models.StatusCompleted, "updatedAt": time.Now()},
	})
	if err != nil {
		log.Printf("Failed to update job %s to completed: %v\n", jobID.Hex(), err)
		return
	}

	log.Printf("Completed job: %s\n", jobID.Hex())
}


func (jw *JobWorker) updateStatus(jobID primitive.ObjectID, status string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jw.jobsCol.UpdateOne(ctx,
		bson.M{"_id": jobID},
		bson.M{
			"$set": bson.M{
				"status":    status,
				"updatedAt": time.Now(),
			},
		},
	)
}

func (jw *JobWorker) EnqueueJob(jobID primitive.ObjectID) {
	jw.jobQueue <- jobID
}

func (jw *JobWorker) Stop() {
	log.Println("Stopping all workers...")
	close(jw.stopChan)
}