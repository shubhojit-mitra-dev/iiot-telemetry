# Product Requirements Document (PRD)

## Project Title: Real-Time Industrial Telemetry & Predictive Maintenance Platform
**Version:** 2.0.0 (Staff Engineer / Solutions Architect Revision)
**Target Timeline:** 12 Hours
**Core Paradigm:** Bifurcated Streaming-Ingest to Serverless Lakehouse (Medallion Architecture + ELT)

---

## 1. Executive Summary & Objective
The goal is to build a production-grade, highly scalable Industrial IoT (IIoT) telemetry platform. The platform handles real-time time-series buffering (Hot Path) for sub-second anomaly detection and active dashboard updates, while simultaneously pushing data down a serverless ETL/ELT pipeline (Cold Path) for schema resolution and batch reporting.

By implementing this, we demonstrate mastery of:
- High-throughput distributed systems (Go).
- Modern Data Engineering (ELT, Medallion Architecture, Parquet).
- Event-driven cloud infrastructure (AWS Serverless).
- DevOps & IaC (Terraform, CI/CD, Observability).
- Practical AI Integration (LLM-based real-time diagnostics).

---

## 2. System Architecture: The Medallion Lakehouse Hybrid

We are utilizing a **Streaming-Ingest to Serverless Lakehouse** architecture. 

*   **Ingestion:** Real-time stream capture over HTTP using a highly concurrent Go backend.
*   **Hot Path (Speed Layer):** Data is parsed in-memory. Latest state is persisted to Redis. Anomalies trigger instant AI diagnostics and broadcast to React clients via WebSockets.
*   **Cold Path (Lakehouse Layer):** Data is streamed to Amazon Kinesis Firehose, which buffers and dumps raw JSON into an S3 "Bronze" layer. AWS Glue infers schema and transforms data for Athena SQL querying.

### 2.1 AWS Services Utilized
*   **Compute:** Amazon ECS (Fargate) + Application Load Balancer (ALB)
*   **Hot Storage:** Amazon ElastiCache (Redis)
*   **Cold Ingestion:** Amazon Kinesis Data Firehose
*   **Data Lake (Bronze/Silver):** Amazon S3
*   **Analytics / ETL:** AWS Glue (Crawler + Data Catalog) & Amazon Athena
*   **Alerting & Observability:** Amazon SNS, Amazon CloudWatch (Logs & Alarms)
*   **Frontend Hosting:** Amazon S3 + CloudFront (or Vercel for rapid prototyping)
*   **Infrastructure as Code:** Terraform
*   **CI/CD:** GitHub Actions

---

## 3. End Product Vision
What the interviewer will see and interact with:
1.  **The "Command Center" Dashboard (React/Vite):** A dark-mode UI displaying a fleet of 10 virtual industrial machines.
2.  **Live Telemetry Streams:** Flowing charts showing Temperature and Vibration metrics updating in real-time without browser lag.
3.  **Instant AI Anomaly Alerts:** When a machine spikes above 120°C, the UI flashes red, and an AI-generated diagnostic message (via OpenRouter/Claude/Llama) appears instantly explaining the likely mechanical failure.
4.  **The Data Lake Query:** A demonstration in the AWS Console showing the raw data landing in S3, the Glue Catalog schema, and a successful Athena SQL query calculating historical rolling averages.

---

## 4. Execution Phases (12-Hour Roadmap)

### Phase 1: Core Ingestion Backend (Hours 1-3)
*   **Focus:** High-performance data capture and real-time state management.
*   **Tasks:**
    *   Initialize Go Fiber/Gin API.
    *   Create `POST /api/v1/telemetry` endpoint.
    *   Implement ElastiCache (Redis) connection to store `device:latest:<id>`.
    *   Implement WebSocket upgrader and broadcast hub.
*   **Expected Output:** A local Go server that can receive JSON, update Redis, and broadcast to a local WebSocket test client.

### Phase 2: AI Integration & Edge Simulator (Hours 3-5)
*   **Focus:** Traffic generation and AI diagnostics.
*   **Tasks:**
    *   Write `simulator.go`: Spawns 10 goroutines sending 50ms interval data with a 2% chance of an anomaly (Temp > 120).
    *   Integrate OpenRouter API in the Backend: When anomaly detected, fetch a 2-sentence diagnostic and broadcast to UI.
    *   Integrate Amazon Kinesis Firehose SDK in the Backend (Cold path drop-off).
*   **Expected Output:** A functioning simulator blasting data at the Go server. The Go server correctly identifies anomalies, fetches AI insights, and pushes to Firehose.

### Phase 3: Infrastructure as Code (Hours 5-7)
*   **Focus:** Cloud provisioning using DevOps best practices.
*   **Tasks:**
    *   Write Terraform (`main.tf`, `variables.tf`, etc.).
    *   Provision VPC, Subnets, ECS Fargate, ALB, Security Groups.
    *   Provision ElastiCache Redis, Kinesis Firehose, S3 Bucket (`telemetry-bronze-lake`).
    *   Provision CloudWatch Log Groups and SNS Topic.
*   **Expected Output:** A fully functional, empty AWS infrastructure ready to receive code, provisioned via a single `terraform apply`.

### Phase 4: Frontend Development (Hours 7-9)
*   **Focus:** The visible showcase.
*   **Tasks:**
    *   Initialize React + Vite + TailwindCSS project.
    *   Implement Recharts for live time-series canvas.
    *   Implement WebSocket consumer to update state arrays (capped at 50 elements to prevent memory leaks).
    *   Build the Anomaly Alert Banner + AI Diagnostic display.
*   **Expected Output:** A polished, fully responsive React application running locally, connecting to the Go backend and displaying live simulator data.

### Phase 5: CI/CD & Cloud Deployment (Hours 9-10)
*   **Focus:** Automating the release process.
*   **Tasks:**
    *   Write `.github/workflows/deploy.yml`.
    *   Automate Docker build and push to Amazon ECR.
    *   Automate ECS task definition update and service restart.
    *   Deploy Frontend to Vercel (or S3/CloudFront).
*   **Expected Output:** The system is live on the internet. The Go simulator running on your local machine is successfully hitting the public AWS ALB endpoint, and the public React URL shows the live data.

### Phase 6: Data Engineering & Final Polish (Hours 10-12)
*   **Focus:** The Medallion Lakehouse proof and interview prep.
*   **Tasks:**
    *   Go to AWS Console -> AWS Glue. Run a Crawler over the `telemetry-bronze-lake` S3 bucket.
    *   Verify the schema in the Data Catalog.
    *   Open Amazon Athena. Write and save 2-3 complex SQL queries (e.g., aggregations, standard deviations).
    *   End-to-end testing, UI polish, and preparing the defense narrative.
*   **Expected Output:** A fully finished project. S3 contains partitioned data, Athena can query it, the dashboard is flawless, and AI alerts are functioning. Ready for the interview.
