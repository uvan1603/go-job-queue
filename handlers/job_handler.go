package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"jobqueue/models"
	"jobqueue/services"
)

// JobHandler handles HTTP requests for job operations
type JobHandler struct {
	jobsCol *mongo.Collection
	worker  *services.JobWorker
}

// NewJobHandler creates a new job handler
func NewJobHandler(jobsCollection *mongo.Collection, jobWorker *services.JobWorker) *JobHandler {
	return &JobHandler{
		jobsCol: jobsCollection,
		worker:  jobWorker,
	}
}

// CreateJobRequest represents the request body for creating a new job
type CreateJobRequest struct {
	Type    string `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// CreateJob handles POST /jobs - creates a new job and enqueues it
func (jh *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" || len(req.Payload) == 0 {
		http.Error(w, "Type and payload are required", http.StatusBadRequest)
		return
	}

	// Create job document
	job := models.Job{
		ID:        primitive.NewObjectID(),
		Type:      req.Type,
		Payload:   req.Payload,
		Status:    models.StatusPending,
		RetryCount: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Insert job into MongoDB
	result, err := jh.jobsCol.InsertOne(ctx, job)
	if err != nil {
		log.Printf("Failed to insert job: %v\n", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	// Enqueue the job for processing
	jobID := result.InsertedID.(primitive.ObjectID)
	jh.worker.EnqueueJob(jobID)

	// Return the created job
	job.ID = jobID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// GetJob handles GET /jobs/{id} - retrieves a job by ID
func (jh *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from URL
	jobIDStr := r.URL.Query().Get("id")
	if jobIDStr == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Convert string to ObjectID
	jobID, err := primitive.ObjectIDFromHex(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID format", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch job from MongoDB
	var job models.Job
	err = jh.jobsCol.FindOne(ctx, bson.M{"_id": jobID}).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Job not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to find job: %v\n", err)
			http.Error(w, "Failed to retrieve job", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// ListJobs handles GET /jobs - lists all jobs (limit 50)
func (jh *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query jobs with limit of 50, sorted by creation date descending
	opts := options.Find().SetLimit(50).SetSort(bson.M{"createdAt": -1})
	cursor, err := jh.jobsCol.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Printf("Failed to query jobs: %v\n", err)
		http.Error(w, "Failed to retrieve jobs", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var jobs []models.Job
	if err = cursor.All(ctx, &jobs); err != nil {
		log.Printf("Failed to decode jobs: %v\n", err)
		http.Error(w, "Failed to retrieve jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}
