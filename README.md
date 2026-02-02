# Job Queue Service

A simple **Job Queue microservice** in Go with **MongoDB** for persistence and **worker-based job processing** with retries.

---

## Features

- **Create jobs** with type and payload
- **Retrieve a job** by ID
- **List recent jobs** (latest 50)
- **Asynchronous processing** via workers
- **Retry mechanism** for failed jobs (configurable)
- **Job statuses:** `pending`, `processing`, `completed`, `failed`
- **Graceful queue handling** with multiple workers

---

## Tech Stack

- **Language:** Go
- **Database:** MongoDB
- **Packages:**
  - `go.mongodb.org/mongo-driver` for MongoDB
  - `encoding/json` for request/response handling
  - Standard Go libraries for concurrency (`channels`, `goroutines`)

---

## Getting Started

### Prerequisites

- Go >= 1.21
- MongoDB running locally or remotely

### Installation

```bash
git clone <repo-url>
cd go-job-queue
go mod tidy
```

Running the Service

```bash
go run cmd/main.go
```

The server starts on http://localhost:8080.

### API Endpoints

1. Create Job
   URL: /jobs

Method: POST

Body:

```bash
{
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome",
    "body": "Hello!"
  }
}
```

Response: 201 Created

```bash
{
  "id": "64f2c2a9c9d2f1a1b2c3d4e5",
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome",
    "body": "Hello!"
  },
  "status": "pending",
  "retryCount": 0,
  "createdAt": "2026-02-02T05:30:00Z",
  "updatedAt": "2026-02-02T05:30:00Z"
}
```

2. Get Job
   URL: /jobs?id=<jobID>

Method: GET

Response:

```bash
{
  "id": "64f2c2a9c9d2f1a1b2c3d4e5",
  "type": "email",
  "payload": { "to": "user@example.com" },
  "status": "completed",
  "retryCount": 1,
  "createdAt": "2026-02-02T05:30:00Z",
  "updatedAt": "2026-02-02T05:32:00Z"
}
```

3. List Jobs
   URL: /jobs

Method: GET

Response: Array of last 50 jobs (descending by createdAt)

### Worker & Retry Mechanism

Multiple worker goroutines process jobs concurrently
Jobs transition through statuses:
pending -> processing -> completed/failed

### Retries:

Max retries configurable (MaxRetries = 3)
Job is re-queued if failed
retryCount tracks attempts
Failed jobs after max retries are marked failed

### Scaling & Overload

Increase numWorkers to process more jobs concurrently
Queue buffer (queueSize) prevents dropping jobs under load
Handles transient failures gracefully with retries

Example MongoDB Document

```bash
{
  "_id": "64f2c2a9c9d2f1a1b2c3d4e5",
  "type": "email",
  "payload": { "to": "user@example.com", "subject": "Welcome" },
  "status": "pending",
  "retryCount": 0,
  "createdAt": "2026-02-02T05:30:00Z",
  "updatedAt": "2026-02-02T05:30:00Z"
}
```

### Future Improvements

Support different job types with custom handlers
Persist job execution logs
Add graceful shutdown for workers
Implement priority queue for urgent jobs

Postman Collection
You can import this directly in Postman:

```bash
{
  "info": {
    "name": "Job Queue Service",
    "_postman_id": "12345-67890-jobqueue",
    "description": "Postman collection for testing Job Queue service",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Create Job",
      "request": {
        "method": "POST",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"type\": \"email\",\n  \"payload\": {\n    \"to\": \"user@example.com\",\n    \"subject\": \"Welcome\",\n    \"body\": \"Hello!\"\n  }\n}"
        },
        "url": {
          "raw": "http://localhost:8080/jobs",
          "protocol": "http",
          "host": ["localhost"],
          "port": "8080",
          "path": ["jobs"]
        }
      }
    },
    {
      "name": "Get Job",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:8080/jobs?id=<jobID>",
          "protocol": "http",
          "host": ["localhost"],
          "port": "8080",
          "path": ["jobs"],
          "query": [
            {
              "key": "id",
              "value": "<jobID>"
            }
          ]
        }
      }
    },
    {
      "name": "List Jobs",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:8080/jobs",
          "protocol": "http",
          "host": ["localhost"],
          "port": "8080",
          "path": ["jobs"]
        }
      }
    }
  ]
}
```
