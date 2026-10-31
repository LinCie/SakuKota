package storagecleanup

// Deletion is a storage object queued for deletion, and the number of failed
// attempts to delete it so far.
type Deletion struct {
	Key      string `db:"key"`
	Attempts int    `db:"attempts"`
}
