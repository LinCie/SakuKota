package storagecleanup

import "time"

// batchSize is the number of due deletions a run claims.
const batchSize = 100

// A failed delete is retried after baseRetryDelay, doubling with each further
// failure up to maxRetryDelay.
const (
	baseRetryDelay = 15 * time.Minute
	maxRetryDelay  = 24 * time.Hour
)

// claimLease is how long a claimed deletion is held back from other workers. A
// worker that stops mid-run leaves its deletions due again after this.
const claimLease = 15 * time.Minute
