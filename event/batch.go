package event

// Batch holds a collection of logs fetched together, along with the block range they span.
type Batch struct {
	Logs      []Log
	FromBlock uint64
	ToBlock   uint64
}

// Len returns the number of logs in the batch.
func (b Batch) Len() int {
	return len(b.Logs)
}

// IsEmpty reports whether the batch contains no logs.
func (b Batch) IsEmpty() bool {
	return len(b.Logs) == 0
}
