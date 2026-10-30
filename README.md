# ChaanBin

ChaanBin is a semantic log search service. It stores log messages and their
embeddings in PostgreSQL, then combines vector similarity with time and
severity filters to return the most relevant logs.

## Why the design

- **PostgreSQL + pgvector** keeps log metadata and embeddings in one queryable
  store.
- **HNSW** provides approximate nearest-neighbor search for low-latency
  similarity queries.
- **Time-based partitions** keep the log table manageable as it grows and
  allow PostgreSQL to prune partitions for time-bounded searches.
- **Ollama** provides local query embeddings using `nomic-embed-text`, so
  search does not require an external embedding service.

The application currently exposes the search API. Log ingestion is represented
by the repository and pipeline packages but is not exposed as an HTTP endpoint.

## Requirements

- Go 1.25.5 or later
- PostgreSQL with the `vector` extension installed
- Ollama with the `nomic-embed-text` model

The schema uses `halfvec(768)` and HNSW indexes, so use a pgvector version
that supports both features.

## Setup

Create a database and enable pgvector:

```sql
CREATE DATABASE chaanbin;
\c chaanbin
CREATE EXTENSION IF NOT EXISTS vector;
```

Apply migrations in order:

```bash
psql "$DATABASE_URL" -f migrations/001_init_schema.sql
psql "$DATABASE_URL" -f migrations/002_create_log_partitions.sql
```

The current partition migration creates October and November 2026 partitions.
Create the next monthly partition before its month begins; inserts without a
matching partition will fail. Keep partition creation in a deployment or
scheduled database-maintenance process as the retention window advances.

Install and start Ollama, then download the embedding model:

```bash
ollama pull nomic-embed-text
ollama serve
```

Set the required database connection string:

```bash
export DATABASE_URL='postgres://user:password@localhost:5432/chaanbin?sslmode=disable'
export PORT=8080
```

Start the service:

```bash
go run ./cmd/chaanbin
```

The server listens on `http://localhost:8080` by default.

## Search API

`POST /api/v1/search` generates an embedding for the query and searches
`app_logs`.

```bash
curl -sS http://localhost:8080/api/v1/search \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "database connection timeout",
    "severity": "error",
    "days_lookback": 7,
    "limit": 10
  }'
```

Request fields:

- `query` — text to search for.
- `severity` — optional exact severity filter.
- `days_lookback` — optional lookback window; defaults to `7`.
- `limit` — optional result limit from `1` to `100`; defaults to `10`.

Each result contains the log ID, service, severity, message, timestamp, and
cosine-based similarity score.

## Development

Run the test suite:

```bash
go test ./...
```

The API tests use fakes for the database repository and embedding client, so
they do not require PostgreSQL or Ollama.
