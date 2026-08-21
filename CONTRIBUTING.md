# Contributing to E-commerce Analytics Engine

Thank you for your interest in contributing! We welcome bug fixes, documentation improvements, and feature contributions.

## Development Setup

### Prerequisites
- Go 1.22+
- Node.js 20+ & npm
- Git

### Backend Setup
```bash
cd backend
go mod download
go test -v ./...
go run cmd/server/main.go
```

### Frontend Setup
```bash
cd frontend
npm ci
npm run lint
npx vitest run
npm run dev
```

## Pull Request Guidelines

1. **Create a branch** from `main` with a descriptive name (`feat/...`, `fix/...`).
2. **Follow Coding Standards**:
   - Keep Go code formatted with `gofmt` and vetted with `go vet`.
   - Keep TypeScript types strict and clean (`npm run lint`).
   - Write comprehensive unit tests for any new features or bug fixes.
3. **Verify locally**:
   - Run `go test -v ./...` in `backend`.
   - Run `npx vitest run` and `npm run build` in `frontend`.
4. **Submit PR**: Provide a clear description referencing related issue numbers.

## License
By contributing, you agree that your contributions will be licensed under the project''s MIT License.
