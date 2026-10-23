package models

type LogEntry struct {
	ServiceName string
	Severity    string
	Message     string
	TimeStamp   string
	Embedding   []float32
}
