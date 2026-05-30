# Niko Admin

<p align="center">
  <a href="./README.md">简体中文</a> | <strong>English</strong>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" alt="Go Version"></a>
  <a href="https://gin-gonic.com/"><img src="https://img.shields.io/badge/Gin-v1.10-blue?logo=go" alt="Gin"></a>
  <a href="https://gorm.io/"><img src="https://img.shields.io/badge/GORM-v1.26-5c6bc0?logo=go" alt="GORM"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL"></a>
  <a href="https://redis.io/"><img src="https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white" alt="Redis"></a>
  <a href="https://www.docker.com/"><img src="https://img.shields.io/badge/Docker-✓-2496ED?logo=docker&logoColor=white" alt="Docker"></a>
  <a href="https://vite.dev/"><img src="https://img.shields.io/badge/Vite-6.x-646CFF?logo=vite&logoColor=white" alt="Vite"></a>
  <a href="https://chakra-ui.com/"><img src="https://img.shields.io/badge/Chakra--UI-2.x-319795?logo=chakra-ui&logoColor=white" alt="Chakra UI"></a>
  <a href="https://swagger.io/"><img src="https://img.shields.io/badge/Swagger-✓-85EA2D?logo=swagger&logoColor=black" alt="Swagger"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow" alt="License"></a>
  <a href="./docs/PRD.md"><img src="https://img.shields.io/badge/PRD-✓-blue" alt="PRD"></a>
</p>

A full-stack admin scaffold built with **Gin + GORM + PostgreSQL + Redis**. The backend provides robust RBAC, dual-token authentication, chunked file upload, and asynchronous task processing; the frontend is developed with **React + Vite + Chakra UI + TypeScript**, delivering a modern, fluid, and highly polished responsive dashboard template.

---

## 📖 Table of Contents
- [Tech Stack](#-tech-stack)
- [Core Features](#-core-features)
- [System Architecture](#-system-architecture)
- [Quick Start](#-quick-start)
  - [Prerequisites](#prerequisites)
  - [Backend Setup](#backend-setup)
  - [Frontend Setup](#frontend-setup)
- [🐳 Docker Deployment](#-docker-deployment)
  - [Local Development Dependencies](#1-local-development-dependencies-postgresql--redis)
  - [Local Unified Containerized Running](#2-local-unified-containerized-running)
  - [Production Deployment](#3-production-deployment-docker-composeprodyml)
- [Daily Command Reference](#-daily-command-reference)
- [Project Directory Structure](#-project-directory-structure)
- [API Specifications & Response Format](#-api-specifications--response-format)
- [Code Generator (CRUD)](#-code-generator-crud)
- [Configuration Guide](#-configuration-guide)
- [Related Documentation](#-related-documentation)
- [License](#license)

---

## 🛠 Tech Stack

### Backend (Go)
* **Web Framework**: [Gin v1.10+](https://gin-gonic.com/) - High-performance HTTP routing and middleware framework.
* **ORM**: [GORM v1.26+](https://gorm.io/) - Developer-friendly ORM library for Golang.
* **Main Database**: PostgreSQL 16+ - For strong consistency and complex querying.
* **Caching & Session**: Redis 7+ - Caches RBAC permissions, sessions, and rate-limiting states.
* **Task Queue**: [Asynq v0.25+](https://github.com/hibiken/asynq) - Lightweight, Redis-backed asynchronous and delayed task queue.
* **Auth**: golang-jwt v5 - Dual Token (Access & Refresh) authentication with Refresh Token Rotation and reuse detection.
* **Real-time Communication**: gorilla/websocket - Structured push notifications with secure JWT handshake.
* **Hot Reload**: [air v1.51+](https://github.com/air-verse/air) - Listens for file changes and restarts backend instantly.

### Frontend (React)
* **Core Framework**: React 19 + TypeScript
* **Build Tool**: Vite 6.x - Ultra-fast hot module replacement (HMR) and bundling.
* **UI Component Library**: Chakra UI v2 - Modern, accessible, and highly customizable styling system.
* **Routing**: React Router v6
* **Data Table**: TanStack Table v8
* **Internationalization**: i18next & react-i18next

---

## ✨ Core Features

* 🔐 **Secure Authentication**: JWT dual token system (Access + Refresh) featuring Refresh Token Rotation and reuse detection. Password hashing via `bcrypt` (cost=12), sensitive information is scrubbed from logs.
* 👥 **RBAC Permission Control**: Fine-grained user-role-permission management. Checks are intercepted at the middleware layer and cached in Redis for high-performance retrieval.
* 📝 **Audit Logging**: Automatically records all write operations (non-GET), asynchronously writing them to PostgreSQL via Asynq queues to prevent request blockages and ensure audit trails.
* 📂 **File Upload Management**: Built-in chunked uploads, resume-from-break, duplicate file detection (instant upload), and multi-threaded downloading. Supports **Local storage**, **PostgreSQL Large Objects (PG lo)**, and **S3-compatible engines (e.g., MinIO/OSS)**.
* 🌐 **Internationalization (i18n)**: Standardized multi-language translations for API error responses, and complete locale switching (Chinese/English) on the frontend.
* 💬 **WebSocket Push**: Integrated real-time notifications, securely validating JWT tokens during the initial connection handshake.
* ⚙️ **CRUD Code Generator**: Instantly generates full-stack code components (DTO, Handler, Service, Repository, Router) from GORM Models via simple CLI commands.
* 📄 **Swagger Documentation**: Annotation-driven, integrated with the CLI generator, enabling easy API testing at a single endpoint in local development.

---

## 📐 System Architecture

The project adheres to a clean, layered architecture with one-way dependencies:

```txt
Handler (Controller) ──> Service (Business Logic) ──> Repository (Data Access) ──> Model (GORM)
         │                         │
         ▼                         ▼
   DTO (Validate)            Storage (local/pg/oss)
```

1. **Handler**: Handles parameter binding, basic validation (`go-playground/validator`), calls Service layer, and structures response. Contains no business logic.
2. **Service**: Core business logic, transaction management, and optional fine-grained permission validations.
3. **Repository**: Pure GORM database queries. Contains no business logic.
4. **DTO**: Input and output Data Transfer Objects, utilizing GORM/validator tags for runtime validation.

---

## 🚀 Quick Start

### Prerequisites
Ensure you have the following installed on your machine:
* Go 1.23+
* Node.js 18+ (pnpm recommended)
* Docker & Docker Compose
* Make utility

---

### Backend Setup

1. **Navigate to the app directory and initialize**:
   ```bash
   cd app
   make init
   ```
   *This copies `.env.example` to `.env`, starts Docker dependency containers (PostgreSQL & Redis), runs go mod tidy, and executes database migrations.*

2. **Start the development server with air hot reload**:
   ```bash
   make serve
   ```
   *The server runs on `http://localhost:8080` with Swagger Docs available at `http://localhost:8080/swagger/index.html`.*

---

### Frontend Setup

1. **Navigate to the web directory**:
   ```bash
   cd web
   ```

2. **Install dependencies**:
   ```bash
   npm install  # or pnpm install / yarn
   ```

3. **Start the Vite development server**:
   ```bash
   npm run dev
   ```
   *The frontend dashboard runs on `http://localhost:5173`.*

---

## 🐳 Docker Deployment

The project includes a robust Dockerfile and multi-environment docker-compose configurations for instant containerization.

### 1. Local Development Dependencies (PostgreSQL + Redis)
If you only want to run PostgreSQL and Redis in containers while keeping your Go and React environments running locally:
```bash
cd app
make docker-up-deps
```
*This is equivalent to running: `docker-compose up -d postgres redis`*

---

### 2. Local Unified Containerized Running
To run the entire system (including the Go backend, built static frontend dashboard, database, and Redis cache) fully containerized:
```bash
# Run at the project root directory
docker-compose up -d --build
```
* During build, it uses a multi-stage Docker builder to automatically compile and build React static assets inside a Node.js environment, copying the output `dist` directly into the lightweight Alpine runtime image to be hosted by the Go backend service.
* Access endpoints:
  * Application frontend & API backend: `http://localhost:8080`
  * PostgreSQL port: `5432`
  * Redis port: `6379`

---

### 3. Production Deployment (docker-compose.prod.yml)
For production environments, container ports are bound only to the local loopback interface (`127.0.0.1:8080`) by default for security. It is recommended to configure Nginx/Caddy as a reverse proxy.

1. **Configure Environment Variables**:
   Ensure `app/.env` contains your customized secure credentials:
   ```bash
   NIKO_JWT_SECRET=your-production-secure-jwt-key
   NIKO_DB_PASSWORD=your-secure-db-password
   ```

2. **Launch the Production Stack**:
   ```bash
   cd app
   make docker-prod
   ```
   *This starts the production stack from `docker-compose.prod.yml`. Storage volumes for database data, files, and server logs are persistently mounted.*

3. **Run Database Migrations Inside the Container**:
   ```bash
   make docker-migrate
   ```

4. **Shutdown the Stack**:
   ```bash
   make docker-prod-down
   ```

---

## 🛠 Daily Command Reference

All backend commands should be run under the `app/` directory via `make`:

```bash
make serve          # Start dependency services & hot reload server (air)
make dev            # Run hot reload server only (assumes external DB & Redis)
make run            # Run directly with go run
make build          # Build the binary for current platform
make build-linux    # Cross-compile Linux amd64 binary
make swag           # Re-generate Swagger API documentation
make gen            # Run CLI generator (scaffold CRUD files for all models)
make unit-test      # Run unit tests (with -race and coverage profile output)
make lint           # Check code styles using golangci-lint
make migrate        # Execute database schema migrations
make docker-up      # Run all services inside containers (Server, DB, Redis)
make docker-down    # Stop and tear down container services
make clean          # Remove builds and coverage files
make help           # Display all make targets with descriptions
```

---

## 📁 Project Directory Structure

```
niko-admin/
├── app/                      # Go backend root
│   ├── cmd/
│   │   ├── server/           # Main server entrypoint
│   │   ├── migrate/          # Migration utility
│   │   └── gen/              # Code Generator CLI
│   ├── internal/
│   │   ├── config/           # Configurations loader (Viper)
│   │   ├── middleware/       # Gin middleware (Auth, RBAC, CORS, i18n, Logger)
│   │   ├── router/           # Route grouping registrations
│   │   ├── handler/          # HTTP Controllers (Validates request, writes responses)
│   │   ├── service/          # Business logic & Transactions
│   │   ├── repository/       # Data Access Layer (GORM operations)
│   │   ├── model/            # Database schema entities (GORM Models)
│   │   ├── dto/              # Request & Response Data Transfer Objects
│   │   ├── pkg/              # Internal utilities (JWT, Crypto, Uniform responses)
│   │   └── task/             # Asynq tasks registration & handlers
│   ├── pkg/
│   │   ├── gen/              # Generator core engines & templates
│   │   └── storage/          # File storage engines (local, pg lo, S3)
│   ├── configs/              # Static configuration files
│   ├── docs/                 # Auto-generated Swagger documentation files
│   ├── Makefile              # Build and task runner scripts
│   ├── Dockerfile            # Multi-stage Dockerfile
│   └── go.mod
├── web/                      # React frontend root
│   ├── src/                  # React source code
│   ├── public/               # Public assets
│   ├── index.html            # SPA main HTML page
│   ├── vite.config.ts        # Vite configuration
│   └── package.json          # Node dependencies & scripts
├── docs/                     # System architecture & PRD docs
│   └── PRD.md
├── openspec/                 # OpenSpec proposals
├── docker-compose.yml        # Development environment docker-compose file
└── docker-compose.prod.yml   # Production docker-compose file
```

---

## 📝 API Specifications & Response Format

### Uniform JSON Response Format

```go
// Success response (HTTP 200 OK)
response.OK(c, data)

// Failure response (HTTP Error Code)
response.Err(c, errors.New(code, message))

// Paginated list response
response.Page(c, list, total, page, pageSize)
```

The output JSON structure matches the following formats:

* **Success**:
  ```json
  {
    "code": 0,
    "message": "success",
    "data": {
      "id": 1,
      "username": "admin"
    }
  }
  ```
* **Failure** (with multi-language support in error messages):
  ```json
  {
    "code": 10001,
    "message": "Invalid username or password"
  }
  ```
* **Paginated**:
  ```json
  {
    "code": 0,
    "message": "success",
    "data": {
      "list": [],
      "total": 100,
      "page": 1,
      "page_size": 20
    }
  }
  ```

---

## ⚙️ Code Generator (CRUD)

The CLI tool scans GORM Models inside `app/internal/model/` and scaffolds complete CRUD layers:

```bash
cd app

# Scaffolds CRUD code for user model (searches for user.go)
go run cmd/gen/main.go user

# Scaffolds CRUD code for all detected models
go run cmd/gen/main.go --all

# List all models eligible for scaffold generation
go run cmd/gen/main.go --list
```

> 💡 **Tip**: Once generated, simply reference and register the generated routes in `app/internal/router/router.go` to bring the new features live.

---

## ⚙️ Configuration Guide

The application supports loading configuration configurations from **Environment variables, `.env` file, and YAML files** (Environment variables take the highest precedence).

To set up your local development parameters, duplicate `app/.env.example` as `app/.env`:

```ini
NIKO_APP_ENV=dev                      # Deployment environment (dev / test / prod)
NIKO_APP_PORT=8080                     # API service port
NIKO_DB_HOST=localhost                 # PostgreSQL hostname
NIKO_DB_PORT=5432                      # PostgreSQL port
NIKO_DB_USER=postgres                  # PostgreSQL username
NIKO_DB_PASSWORD=postgres              # PostgreSQL password
NIKO_DB_NAME=niko_admin                # PostgreSQL database name
NIKO_REDIS_HOST=localhost              # Redis hostname
NIKO_REDIS_PORT=6379                   # Redis port
NIKO_JWT_SECRET=change-me-in-production# JWT secret key (must be secure in prod)
NIKO_STORAGE_DRIVER=local              # Storage driver (local / pg / oss)
```

---

## 📚 Related Documentation

* [PRD Product Requirements Document](./docs/PRD.md)
* Swagger UI: `http://localhost:8080/swagger/index.html` (Accessible once server is running)

---

## 📄 License

[MIT License](./LICENSE)
