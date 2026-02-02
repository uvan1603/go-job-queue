package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"jobqueue/db"
	"jobqueue/handlers"
	"jobqueue/services"
)

func main() {
	// MongoDB connection URI
	// Set via environment variable or use default
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// Connect to MongoDB
	mongoClient, err := db.ConnectMongoDB(mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(nil)

	// Get jobs collection
	jobsCol := db.GetJobsCollection(mongoClient)

	// Create job worker with buffered channel (capacity 100) and 2 worker goroutines
	// The channel acts as the in-memory queue for job IDs
	jobWorker := services.NewJobWorker(jobsCol, 100, 2)

	// Start worker goroutines to process jobs asynchronously
	jobWorker.Start()

	// Create job handler
	jobHandler := handlers.NewJobHandler(jobsCol, jobWorker)

	// Register HTTP routes
	http.HandleFunc("/jobs", handleJobsRoute(jobHandler))
	http.HandleFunc("/health", handleHealth)

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr: ":" + port,
	}

	log.Printf("Starting server on port %s\n", port)

	// Start server in a goroutine so we can listen for shutdown signals
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received")

	// Stop the job worker
	jobWorker.Stop()

	// Shutdown the HTTP server
	if err := server.Shutdown(nil); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// handleJobsRoute routes requests based on the HTTP method and query parameters
func handleJobsRoute(handler *handlers.JobHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for all endpoints
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			return
		}

		switch r.Method {
		case http.MethodPost:
			handler.CreateJob(w, r)
		case http.MethodGet:
			// Check if job ID is provided in query string
			if r.URL.Query().Get("id") != "" {
				handler.GetJob(w, r)
			} else {
				handler.ListJobs(w, r)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleHealth is a simple health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"healthy"}`)
}
