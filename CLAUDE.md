# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VelaUX is the web portal and UX dashboard for [KubeVela](https://github.com/kubevela/kubevela). It provides an extensible application delivery platform with multi-cluster/multi-environment support, pipeline management, and observability. The project uses a Go backend and a React/TypeScript frontend organized as a Yarn 3 workspace monorepo.

## Build Commands

### Frontend
```shell
yarn install                    # Install dependencies (requires Yarn 2+)
yarn build                      # Build all packages + UI
yarn build-packages             # Build only packages (theme, data, ui, plugins)
yarn dev                        # Start dev server with hot reload
yarn test                       # Run UI tests
yarn lint                       # Lint frontend code
yarn lint:fix                   # Auto-fix lint issues
```

### Backend
```shell
go mod tidy                     # Sync dependencies
make run-server                 # Start server (requires KubeVela environment)
make unit-test-server-local     # Run server unit tests (Mac-compatible)
make unit-test-server-ci        # Run server unit tests (CI environment)
make e2e-server-test            # Run e2e tests
make test-db-up                 # Start test databases with docker-compose
make test-db-down               # Stop test databases
```

### Combined / Docker
```shell
make build-ui                   # Build frontend only
make docker-build               # Build Docker image
make build-test-image           # Build UI + Docker image for local testing
```

### Code Quality
```shell
make reviewable                 # Run: mod, fmt, vet, staticcheck, lint
make check-diff                 # Run reviewable + ensure branch is clean
make build-swagger              # Generate OpenAPI schema
```

### Environment Setup
```shell
make setup-test-server          # Install kubebuilder and envtest tools
vela addon enable ./addon       # Enable VelaUX addon in KubeVela
```

## Architecture

### Go Backend (`pkg/server/`)

Clean Architecture with these layers:
- **interfaces/api/** - REST API handlers, DTOs (`dto/v1/`), and Assemblers (`assembler/v1/`)
- **domain/** - Business logic organized as:
  - `model/` - Database entity models
  - `repository/` - Data access interfaces and implementations
  - `service/` - Domain services (50+ files for different domains)
- **event/** - Asynchronous task workers (master node only)
- **infrastructure/** - Technical foundations (database, cache, kube client, MQ)

Key entry point: `cmd/server/main.go`

### Frontend (`packages/velaux-ui/src/`)

React/TypeScript UI organized as:
- `api/` - API client definitions
- `components/` - Reusable UI components
- `pages/` - Page-level components (36 subdirectories)
- `model/` - Frontend data models/state
- `extends/` - Extended functionality
- `layout/` - App layout components
- `utils/` - Utility functions
- `services/` - Service layer

### Packages (Yarn workspaces)
- `packages/velaux-ui/` - Main React application
- `packages/velaux-theme/` - Theme/styling
- `packages/velaux-data/` - Data utilities
- `plugins/*/` - Plugin examples (app-demo, node-dashboard)

### Other Key Directories
- `addon/` - KubeVela addon definition for VelaUX
- `test/` - Integration tests using kubebuilder/ginkgo
- `e2e-test/` - E2E test fixtures and CRDs

## Important Notes

- **Yarn 2+ is required** for frontend development
- **Go 1.19+** is required for backend
- Server must connect to a KubeVela control plane (use `velad install`)
- The `pkg/server/interfaces/api/dto/v1/types.go` file contains many DTO type definitions
- Tests in `pkg/server/domain/service` must run serially with `ginkgo --procs=1 --focus="serial"` (see `make unit-test-server-ci`)
- VSCode should configure TypeScript SDK from `.yarn/sdks/typescript/lib` when using Yarn 2 PnP

## Tech Stack

- **Frontend**: React 17, TypeScript, @alifd/next (Fusion Design), Webpack 5
- **Backend**: Go, chi router, GORM, kubebuilder
- **Testing**: Ginkgo, Cypress (e2e), Jest (UI)
- **Infrastructure**: Kubernetes, KubeVela core
