package ingest

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/abhiraj-ku/ChaanBin/internal/models"
	"github.com/abhiraj-ku/ChaanBin/internal/store"
)

type Pipeline struct {
	repo         *store.LogRepository
	workerCount  int
	batchSize    int
	flushTimeout time.Duration
}

func NewPipeline(repo *store.LogRepository, workers, batchSize int, timeout time.Duration) *Pipeline {
	return &Pipeline{
		repo:         repo,
		workerCount:  workers,
		batchSize:    batchSize,
		flushTimeout: timeout,
	}
}

func (p *Pipeline) Start(ctx context.Context, rawLogs <-chan models.LogEntry) {
	processedLogs := make(chan models.LogEntry, p.batchSize*2)
	var wg sync.WaitGroup

	// start the workers
	for i := 0; i < p.workerCount; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			p.runWorker(ctx, rawLogs, processedLogs)
		}(i)
	}

	// start the db flusher in the bg
	flusherDone := make(chan struct{})
	go func() {
		p.runFlusher(ctx, processedLogs)
		close(flusherDone)

	}()

	// let worker finish the jobs
	wg.Wait()

	// signal the flusher to flysh
	close(processedLogs)

	// wait for flusher to completet the final db write
	<-flusherDone
}

func (p *Pipeline) runWorker(ctx context.Context, in <-chan models.LogEntry, out chan<- models.LogEntry) {

	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-in:
			if !ok {
				return // channel is closed
			}
			// TODO: actual http call to embedding models for the logs to get vectorize
			entry.Embedding = p.mockEmbedding(entry.Message)

			select {
			case out <- entry:
			case <-ctx.Done():
				return
			}

		}
	}
}

func (p *Pipeline) runFlusher(ctx context.Context, in <-chan models.LogEntry) {
	batch := make([]models.LogEntry, 0, p.batchSize)
	ticker := time.NewTicker(p.flushTimeout)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := p.repo.BulkInsert(ctx, batch); err != nil {
			log.Printf("DB Write Error: %v\n", err)
			// In production, write failed batches to a Dead Letter Queue (DLQ)
		}
		batch = batch[:0] // Reset slice length but retain capacity
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush() // Flush periodically even if batch size isn't met
		case entry, ok := <-in:
			if !ok {
				flush() // Flush remaining items when channel closes
				return
			}

			batch = append(batch, entry)
			if len(batch) >= p.batchSize {
				flush()
				ticker.Reset(p.flushTimeout) // Reset timer after a full batch
			}
		}
	}

}

func (p *Pipeline) mockEmbedding(msg string) []float32 {
	return make([]float32, 1536)
}
