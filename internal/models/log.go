package models

import "time"

type LogEntry struct {
	ServiceName string
	Severity    string
	Message     string
	TimeStamp   time.Time
	Embedding   []float32
}
