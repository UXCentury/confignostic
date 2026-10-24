# Contributing

Confignostic is an early-stage project. Propose focused use cases before large features. Keep provider integrations behind small, tested interfaces, disclose unsupported remote operations, and never commit credentials.

Backend checks: `go fmt ./...`, `go vet ./...`, `go test ./...`, `go build ./cmd/confignostic-api`.

Frontend checks: `cd web && npm install && npm run build`.
