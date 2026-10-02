# Bank Core API

REST API for a small banking system, written in Go and backed by PostgreSQL.

## Features

- Customers: create, list, get, update type/status/contacts, delete
- Accounts: create, list, get by id or account number, change status
- Transactions: deposit, withdraw, transfer by account number
- Transaction history: all transactions and per-account history

## Requirements

- Go 1.22 or later
- PostgreSQL 12 or later
- `psql` (or another client) to apply the SQL files

## Environment configuration

Copy `.env.example` to `.env` and set a connection string for your local PostgreSQL instance:

```
ENV=development
APP_HOST=127.0.0.1
APP_PORT=4000
DATABASE_URL=postgres://postgres:postgres@localhost:5432/go_bank_core_api?sslmode=disable
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/go_bank_core_api?sslmode=disable
LOG_ROOT_PATH=./logs
LOG_MAX_SIZE=100
LOG_MAX_AGE=24
LOG_MAX_BACKUPS=3
```

`TEST_DATABASE_URL` is optional. Tests fall back to `DATABASE_URL` when it is unset.

## Database setup

From the repository root, apply the SQL files in order. The first script creates the `go_bank_core_api` database.

Linux / macOS:

```bash
psql -U postgres -f _database/1-tables.sql
psql -U postgres -d go_bank_core_api -f _database/2-views.sql
psql -U postgres -d go_bank_core_api -f _database/3-first-inserts.sql
psql -U postgres -d go_bank_core_api -f _database/4-indexes.sql
```

Windows (PowerShell):

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -f _database/1-tables.sql
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d go_bank_core_api -f _database/2-views.sql
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d go_bank_core_api -f _database/3-first-inserts.sql
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d go_bank_core_api -f _database/4-indexes.sql
```

Re-running `1-tables.sql` drops and recreates the database.

Seed accounts after setup:

| Account number | Status   | Opening balance |
|----------------|----------|-----------------|
| `1000000001`   | Active   | 10000.00        |
| `1000000002`   | Active   | 500.00          |
| `2000000001`   | Active   | 2500.00         |
| `2000000002`   | Inactive | 100.00          |

## Run the API

```bash
go mod tidy
go run main.go
```

The server listens on `http://127.0.0.1:4000` by default.

## Example endpoints

Transfer:

```http
POST /api/transactions/transfer
Content-Type: application/json
Idempotency-Key: 0191f3c2-transfer-001

{
  "source_number": "1000000001",
  "destination_number": "2000000001",
  "amount": 250.50,
  "currency": "USD"
}
```

```http
POST /api/accounts
GET  /api/accounts
GET  /api/accounts/{id}
GET  /api/accounts/by-number/{account_number}
PUT  /api/accounts/change-status

POST /api/transactions/deposit
POST /api/transactions/withdraw
GET  /api/transactions
GET  /api/transactions/account/{account_number}
```

A Postman collection lives in `_api_collections/postman.postman_collection.json`.

## Run tests

Tests use a real PostgreSQL database. Apply the schema first, then:

```bash
go test ./...
```

If PostgreSQL is unreachable, transfer tests are skipped.

```bash
gofmt -l .
go vet ./...
```

## Project layout

```
main.go
common/           configuration, errors, HTTP helpers
core/controllers  HTTP routes
core/services     application logic
core/repositories database access
core/entities     models and request types
_database/        SQL schema, views, seed data, indexes
```
