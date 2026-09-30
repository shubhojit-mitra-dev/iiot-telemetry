# Phase 3: Infrastructure as Code & Serverless Lakehouse Ingestion

**Target Audience:** Senior/Staff Implementing Agents, DevOps Engineers, Cloud Architects
**Objective:** Provision a production-ready, highly available AWS environment using Terraform, and implement the "Cold Path" streaming integration in the Go backend to forward telemetry data to Amazon Kinesis Data Firehose.

## 1. Phase 1 & 2 Outcomes & Current State

Before beginning Phase 3, here is a summary of the system state:
*   **High-Throughput Hot Path:** Vanilla Go backend ingests 10,000+ msgs/sec, recycling memory via `sync.Pool`, updating state in Redis (or in-memory fallback), and fanning out via WebSockets.
*   **Resilient WebSocket Hub:** Non-blocking Gorilla WebSocket hub prevents slow clients from stalling the ingestion pipeline.
*   **Edge Simulator:** A robust Go CLI simulator generates realistic physical telemetry for 10 distinct factory machines with multi-tick anomalies and machine-specific load curves.
*   **React Dashboard:** A Vite/React frontend connects via WebSocket, visualizes sliding window chart buffers without memory leaks, and aggregates server-side health stats.
*   **Quality Assurance:** 80%+ test coverage, race-condition free, passing strict CI pipelines.

## 2. Phase 3 Objectives

Phase 3 shifts focus from local runtime to **Cloud Infrastructure and the Medallion Data Architecture**. We will use Terraform to provision the entire AWS stack and update the Go backend to stream raw JSON payloads down the "Cold Path" into Amazon S3 via Kinesis Firehose.

### 2.1 Terraform Infrastructure (IaC)
All cloud resources must be codified using Terraform to ensure reproducible deployments.

*   **Location:** `terraform/`
*   **Components to Provision:**
    *   **Networking:** VPC, 2 Public Subnets, 2 Private Subnets, Internet Gateway, NAT Gateway (for ECS private egress).
    *   **Compute (App Tier):** ECS Cluster, Fargate Task Definition (running the Go backend), and ECS Service.
    *   **Load Balancing:** Application Load Balancer (ALB) listening on port 80/443, routing to ECS Fargate tasks via Target Group.
    *   **Data Tier (Hot Path):** ElastiCache Serverless (Redis) in private subnets with Security Group restricted to ECS tasks.
    *   **Data Tier (Cold Path):** 
        *   S3 Bucket (`<project-prefix>-telemetry-bronze-lake`) for raw JSON storage.
        *   Amazon Kinesis Data Firehose delivery stream configured to buffer (e.g., 5MB or 60s) and dump data into the Bronze S3 bucket.
    *   **Observability:** CloudWatch Log Groups for ECS and Firehose.
    *   **IAM & Security:** Least-privilege IAM Roles for ECS Task Execution, ECS Task (allowing `firehose:PutRecordBatch`), and Firehose (allowing `s3:PutObject`). Strict Security Groups controlling ingress/egress.

### 2.2 Go Backend Cold Path Integration
The ingestion worker must forward telemetry data to Kinesis Data Firehose without blocking the real-time WebSocket broadcast or Redis HSET operations.

*   **Target:** `app/service/worker.go` and a new `app/repository/firehose.go`
*   **Architecture:**
    *   Import `github.com/aws/aws-sdk-go-v2/service/firehose`.
    *   Implement an asynchronous, batched Firehose publisher. Instead of sending 1 HTTP request to AWS per payload, the worker pool should append payloads to a local batch slice.
    *   When the batch hits `500` records or a `1-second` ticker fires, flush the batch via `PutRecordBatch` to the Kinesis delivery stream.
    *   This ensures high throughput while staying well within AWS API rate limits and minimizing network overhead.
    *   If Firehose is unavailable (e.g., missing credentials in local dev mode), the backend must gracefully bypass the cold path without crashing, continuing to serve the hot path.

## 3. Implementation Steps & Commit Plan

Following the strict rules (Max 2 files per commit, atomic commits):

1.  **Commit 1:** Create `terraform/vpc.tf` and `terraform/security.tf` (Networking & SGs).
2.  **Commit 2:** Create `terraform/ecs.tf` and `terraform/alb.tf` (Compute & Load Balancing).
3.  **Commit 3:** Create `terraform/data.tf` and `terraform/iam.tf` (Redis, S3, Firehose, Roles).
4.  **Commit 4:** Create `terraform/variables.tf` and `terraform/outputs.tf`.
5.  **Commit 5:** Update `go.mod`/`go.sum` to include AWS SDK v2, and create `app/repository/firehose.go` for batched publishing.
6.  **Commit 6:** Update `app/service/worker.go` and `main.go` to wire the Firehose publisher into the asynchronous ingestion loop.

## 4. Success Criteria for Phase 3
*   Running `terraform init`, `terraform plan`, and `terraform apply` successfully provisions the full AWS infrastructure from a blank state.
*   The Go backend seamlessly initializes the AWS SDK based on environment variables or IAM roles.
*   The `TestIngestionService_ProcessAndStats` and other unit tests continue to pass (with a mock Firehose publisher injected).
*   During a local load test (`go run cmd/bench/main.go -rate=5000`), the Go backend successfully batches and flushes records to Kinesis Firehose in the background.
*   The S3 Bronze bucket eventually populates with raw JSON files partitioned by Year/Month/Day/Hour.
