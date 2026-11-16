# Cost demo — CRM (Go)

A basic CRM for adding and removing customers (first name, last name, email). The Go counterpart
of `example-typescript`, `example-java` and `example-python`: same REST API, same SQLite schema, same page.

- `backend/` — REST API (`net/http`) storing data in a local SQLite file via `database/sql` and the
  pure-Go driver `modernc.org/sqlite` (no cgo).
- `frontend/` — Go web server (standard library only) that renders the page with `html/template` by
  calling the API, and proxies `/api` to it.

Requires Go 1.22 or newer (it uses the method and wildcard patterns of `http.ServeMux`).

## Setup

```sh
(cd backend && go mod tidy)          # fetches the SQLite driver and writes go.sum
(cd backend && go run ./cmd/seed)    # optional demo customers
```

## Run

In two terminals:

```sh
cd backend && PORT=4000 go run ./cmd/api
cd frontend && PORT=5173 API_PORT=4000 go run ./cmd/web
```

| Variable        | Used by  | Default                         | Purpose                                 |
| --------------- | -------- | ------------------------------- | --------------------------------------- |
| `PORT`          | both     | `4000` backend, `5173` frontend | Port the server listens on              |
| `DATABASE_PATH` | backend  | `data/crm.sqlite` (in `backend/`) | SQLite file (ignored by git)          |
| `API_PORT`      | frontend | `4000`                          | Backend port the frontend calls         |
| `API_URL`       | frontend | `http://localhost:$API_PORT`    | Full backend URL (overrides `API_PORT`) |

## Test

```sh
(cd backend && go test ./...)
(cd frontend && go test ./...)
```

## API

| Method   | Path                  | Body                               | Responses                     |
| -------- | --------------------- | ---------------------------------- | ----------------------------- |
| `GET`    | `/api/customers`      | —                                  | `200` list of customers       |
| `POST`   | `/api/customers`      | `{firstName, lastName, email}`     | `201`, `400` invalid, `409` duplicate email |
| `DELETE` | `/api/customers/:id`  | —                                  | `204`, `404`                  |

The frontend serves the same `/api` routes by proxying them to the backend.

## Pages (frontend)

| Method | Path                    | Does                                                                                  |
| ------ | ----------------------- | ------------------------------------------------------------------------------------- |
| `GET`  | `/`                     | Customer count, add form and table                                                    |
| `POST` | `/customers`            | Adds a customer, then redirects to `/`; on error re-renders the form with the message |
| `POST` | `/customers/:id/delete` | Removes a customer (after a browser confirm), then redirects to `/`                   |
