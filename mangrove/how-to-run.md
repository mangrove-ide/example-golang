# How to run (Mangrove)

- Prerequisites: Go 1.22+. The backend needs `modernc.org/sqlite`: run `go mod tidy` in `backend/` once (it writes `go.sum`). The frontend uses the standard library only.
- Setup command: `cd backend && go mod tidy && go run ./cmd/seed` (the seed adds optional demo data)
- Start the frontend on the grove port: `cd frontend && PORT={{port}} API_PORT=$(( {{port}} + 5 )) go run ./cmd/web`
- Extra process "api": `cd backend && PORT=$(( {{port}} + 5 )) go run ./cmd/api`
- Neither server reloads on change: restart the apps after editing Go code.
- Tests: `cd backend && go test ./...`, `cd frontend && go test ./...`.
