package store

import (
	"context"
	"fmt"
	"time"

	"github.com/abhiraj-ku/ChaanBin/internal/db"
	"github.com/abhiraj-ku/ChaanBin/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

type LogRepository struct {
	db *db.Database
}
type SearchParams struct {
	QueryVector  []float32
	Severity     *string
	DaysLookback int
	Limit        int
}

type SearchResults struct {
	ID         int64     `json:"id"`
	Service    string    `json:"service"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	TimeStamp  time.Time `json:"timestamp"`
	Similarity float64   `json:"similarity"`
}

func NewLogRepository(db *db.Database) *LogRepository {
	return &LogRepository{db: db}
}

// send batch query at once in one network trip
func (r *LogRepository) BulkInsert(ctx context.Context, logs []models.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	query := `
		insert into app_logs (
			service_name,
			severity,
			message,
			ingested_at,
			embedding
		)
		values ($1, $2, $3, $4, $5)`

	for _, log := range logs {
		batch.Queue(query, log.ServiceName, log.Severity, log.Message, log.TimeStamp, log.Embedding)
	}

	br := r.db.Pool.SendBatch(ctx, batch)
	defer br.Close()

	// send the batch but need to make suer they all execute sucesfully
	for i := 0; i < len(logs); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert the batch row %d: %w", i, err)
		}
	}

	return nil
}

// find the similar vector to search params
func (r *LogRepository) SearchSimilarLogs(ctx context.Context, params SearchParams) ([]SearchResults, error) {

	query := `
		select
			id,
			service_name,
			severity,
			message,
			ingested_at,
			1 - (embedding <=> $1) as similarity
		from app_logs
		where ($2::text is null or severity = $2)
		and ingested_at >= now() - ($3::int || ' days')::interval
		order by embedding <=> $1
		limit $4
	`

	vec := pgvector.NewVector(params.QueryVector)

	rows, err := r.db.Pool.Query(ctx, query, vec, params.Severity, params.DaysLookback, params.Limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var results []SearchResults

	for rows.Next() {
		var res SearchResults

		if err := rows.Scan(&res.ID, &res.Service, &res.Severity, &res.Message, &res.TimeStamp, &res.Similarity); err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
