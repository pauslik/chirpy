# Chirpy

Simple chirp service (Go). See the code in [main.go](main.go).

---

## Overview

Chirpy is a small web service that supports:
- User signup, login (JWT + refresh tokens)
- Creating / listing / deleting "chirps"
- Admin metrics & reset endpoints
- A webhook endpoint to upgrade users (Polka)

Core types and handlers live in:
- server bootstrap: [main.go](main.go) (`apiConfig`)  
- user endpoints: [`apiConfig.createUserHandler`](handlers_users.go), [`apiConfig.changeUserHandler`](handlers_users.go), [`apiConfig.loginHandler`](handlers_users.go), [`apiConfig.refreshHandler`](handlers_users.go), [`apiConfig.revokeHandler`](handlers_users.go)  
- chirps endpoints: [`apiConfig.createChirpHandler`](handlers_chirps.go), [`apiConfig.getChirpsHandler`](handlers_chirps.go), [`apiConfig.getChirpIDHandler`](handlers_chirps.go), [`apiConfig.deleteChirpIDHandler`](handlers_chirps.go)  
- admin: [`apiConfig.metricsHandler`](handlers_admin.go), [`apiConfig.resetHandler`](handlers_admin.go)  
- polka webhook: [`apiConfig.upgradeRedHandler`](handlers_polka.go)  
- middleware: [`apiConfig.middlewareMetricsInc`](middleware.go)

Auth helpers in [`internal/auth`](internal/auth):
- JWT and refresh token: [`MakeJWT`](internal/auth/tokens.go), [`ValidateJWT`](internal/auth/tokens.go), [`MakeRefreshToken`](internal/auth/tokens.go)  
- Password helpers and header parsing: [`HashPassword`](internal/auth/auth.go), [`CheckPasswordHash`](internal/auth/auth.go), [`GetBearerToken`](internal/auth/auth.go), [`GetAPIKey`](internal/auth/auth.go)

Database access is generated with sqlc into [`internal/database`](internal/database):
- Users: [`CreateUser`](internal/database/users.sql.go), [`GetUser`](internal/database/users.sql.go), [`UpdateUser`](internal/database/users.sql.go), [`UpgradeUserRed`](internal/database/users.sql.go), [`ResetUsers`](internal/database/users.sql.go)  
- Refresh tokens: [`CreateRefreshToken`](internal/database/refresh_tokens.sql.go), [`GetRefreshToken`](internal/database/refresh_tokens.sql.go), [`GetUserFromRefreshToken`](internal/database/refresh_tokens.sql.go), [`RevokeRefreshToken`](internal/database/refresh_tokens.sql.go)  
- Chirps: [`CreateChirp`](internal/database/chirps.sql.go), [`GetChirp`](internal/database/chirps.sql.go), [`GetChirps`](internal/database/chirps.sql.go), [`GetChirpsUser`](internal/database/chirps.sql.go), [`DeleteChirp`](internal/database/chirps.sql.go), [`ResetChirps`](internal/database/chirps.sql.go)

SQL schema and queries:
- schema files: [sql/schema](sql/schema)  
- queries: [sql/queries](sql/queries)  
- sqlc config: [sqlc.yaml](sqlc.yaml)

---

## Environment

Required env vars (can be loaded via `.env` and `godotenv` is used in [main.go](main.go)):
- DB_URL — Postgres connection string
- JWT_SECRET — secret for signing JWTs
- POLKA_KEY — API key for Polka webhook verification

`.env` is in `.gitignore`.

---

## Build & Run

1. Prepare Postgres and run DB migrations using your preferred tool (schema files are in `sql/schema`).
2. Build / run:
```sh
go run .
# or
go build -o chirpy ./...
./chirpy
```

The server listens on `:8080` by default (see [main.go](main.go)).

---

## Tests

Auth tests:
- [`internal/auth/auth_test.go`](internal/auth/auth_test.go)
- [`internal/auth/tokens_test.go`](internal/auth/tokens_test.go)

Run all tests:
```sh
go test ./...
```

---

## Endpoints

Note: handlers reference Bearer tokens and API keys using the functions in [`internal/auth`](internal/auth).

- GET /api/healthz  
  - Description: Health check  
  - Response: 200 "OK" (plain text)

- Static files
  - GET /app/* — serves files from repo root (`/app/` uses `http.FileServer`)
  - GET /app/assets — serves `./app/assets`  
  - Middleware increments visit count via [`apiConfig.middlewareMetricsInc`](middleware.go)

- Admin
  - GET /admin/metrics  
    - Handler: [`apiConfig.metricsHandler`](handlers_admin.go)  
    - Response: 200 HTML with visit count
  - POST /admin/reset  
    - Handler: [`apiConfig.resetHandler`](handlers_admin.go)  
    - Action: resets visits and clears users / chirps via [`ResetUsers`](internal/database/users.sql.go) and [`ResetChirps`](internal/database/chirps.sql.go)  
    - Response: 200 "OK\n"

- Users
  - POST /api/users  
    - Handler: [`apiConfig.createUserHandler`](handlers_users.go)  
    - Request JSON:
      ```json
      { "email": "user@example.com", "password": "plaintext" }
      ```
    - Response 201 JSON: created user fields
      - keys: id, created_at, updated_at, email, is_chirpy_red
  - PUT /api/users  
    - Handler: [`apiConfig.changeUserHandler`](handlers_users.go)  
    - Auth: Authorization: Bearer <JWT> (use [`GetBearerToken`](internal/auth/auth.go) and validate with [`ValidateJWT`](internal/auth/tokens.go))  
    - Request JSON:
      ```json
      { "email": "new@example.com", "password": "newpass" }
      ```
    - Response 200 JSON: updated user fields
  - POST /api/login  
    - Handler: [`apiConfig.loginHandler`](handlers_users.go)  
    - Request JSON:
      ```json
      { "email": "user@example.com", "password": "plaintext" }
      ```
    - Response 200 JSON: user object including `token` (JWT) and `refresh_token` (server-saved token)
      - Creates a refresh token via [`MakeRefreshToken`](internal/auth/tokens.go) and stores via [`CreateRefreshToken`](internal/database/refresh_tokens.sql.go)
    - Errors: 401 on bad creds
  - POST /api/refresh  
    - Handler: [`apiConfig.refreshHandler`](handlers_users.go)  
    - Auth header: Authorization: Bearer <refresh_token>  
    - Action: validates refresh token via [`GetRefreshToken`](internal/database/refresh_tokens.sql.go) and returns a new JWT via [`MakeJWT`](internal/auth/tokens.go)  
    - Response: 200 JSON `{ "token": "<JWT>" }`  
    - Errors: 401 if refresh token not found / revoked
  - POST /api/revoke  
    - Handler: [`apiConfig.revokeHandler`](handlers_users.go)  
    - Auth header: Authorization: Bearer <refresh_token>  
    - Action: revokes the refresh token via [`RevokeRefreshToken`](internal/database/refresh_tokens.sql.go)  
    - Response: 204 (no content)

- Chirps
  - POST /api/chirps  
    - Handler: [`apiConfig.createChirpHandler`](handlers_chirps.go)  
    - Auth: Authorization: Bearer <JWT>  
    - Request JSON:
      ```json
      { "body": "Hello world" }
      ```
    - Constraints: max 140 chars; the body is sanitized for banned words (see `cleanBody` in handlers_chirps.go)  
    - Response: 201 JSON chirp (id, created_at, updated_at, body, user_id)
  - GET /api/chirps  
    - Handler: [`apiConfig.getChirpsHandler`](handlers_chirps.go)  
    - Optional query: `?author_id=<uuid>` to filter by user  
    - Optional query: `?sort=<asc/desc>` to sort by `created_at` value, `asc` is default
    - Response: 200 JSON array of chirps
  - GET /api/chirps/{chirpID}  
    - Handler: [`apiConfig.getChirpIDHandler`](handlers_chirps.go)  
    - Path param: `chirpID` (uuid)  
    - Response: 200 JSON chirp or 404 if not found
  - DELETE /api/chirps/{chirpID}  
    - Handler: [`apiConfig.deleteChirpIDHandler`](handlers_chirps.go)  
    - Auth: Authorization: Bearer <JWT>  
    - Only the author may delete; returns 204 on success, 403 if forbidden, 404 if not found

- Polka webhook
  - POST /api/polka/webhooks  
    - Handler: [`apiConfig.upgradeRedHandler`](handlers_polka.go)  
    - Auth header: Authorization: Bearer <POLKA_KEY> (compare to `apiCfg.polka`) using [`GetAPIKey`](internal/auth/auth.go)  
    - Request JSON shape:
      ```json
      {
        "event": "user.upgraded",
        "data": { "user_id": "<uuid>" }
      }
      ```
    - Action: calls [`UpgradeUserRed`](internal/database/users.sql.go) to set `is_chirpy_red` on the user  
    - Response: 204 on success, 401 on bad API key

---

## Notes & Implementation details

- SQLC is configured in [sqlc.yaml](sqlc.yaml). Generated Go DB code resides in [`internal/database`](internal/database).
- Passwords are hashed with argon2id via [`github.com/alexedwards/argon2id`](internal/auth/auth.go).
- JWT uses [`github.com/golang-jwt/jwt/v5`](internal/auth/tokens.go).
- Refresh tokens stored in `refresh_tokens` table (`sql/schema/004_refresh_tokens.sql`).

---

If you want, I can add example curl commands for each endpoint or scaffolding for DB migrations into the repo.
