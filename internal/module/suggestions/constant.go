package suggestions

// MaxPendingPerUser is the number of suggestions a user may have waiting for review at once.
const MaxPendingPerUser = 5

// Statuses a suggestion can have. A suggestion is reviewed once, so it leaves
// pending for approved or rejected and never changes again.
const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

// Lengths in characters, counted after trimming spaces. The name and optional
// place fields share the limits of place, so an approved suggestion always
// fits the place it becomes.
const (
	maxNoteLength   = 1000
	maxReasonLength = 1000
)
