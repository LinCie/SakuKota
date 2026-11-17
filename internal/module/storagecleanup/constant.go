package storagecleanup

import "time"

// batchSize is the number of due keys a run processes.
const batchSize = 100

// retryDelay is how long a failed delete waits before it is due again.
const retryDelay = time.Hour
