package store

import (
	"context"
	"fmt"

	"github.com/abhiraj-ku/ChaanBin/internals/db"
	"github.com/abhiraj-ku/ChaanBin/internals/models"
	"github.com/jackc/pgx/v5"
)

type LogRepository struct {
	db *db.Database
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
	
		insert into app_log(service_name,severity,message,ingested_at,embedding)
		values($1,$2,$3,$4,$5)`

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
