package suggestion

// MaxPendingPerUser is the number of suggestions a user may have waiting for review at once.
const MaxPendingPerUser = 5

// Statuses a suggestion can have. A suggestion is reviewed once, so it leaves
// pending for approved or rejected and never changes again.
const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

// Lengths in characters, counted after trimming spaces.
const (
	maxNameLength   = 150
	maxNoteLength   = 1000
	maxReasonLength = 1000
)

// Indonesia's bounding box in degrees: west and east longitude, south and north latitude.
const (
	indonesiaMinLongitude = 95
	indonesiaMaxLongitude = 141
	indonesiaMinLatitude  = -11
	indonesiaMaxLatitude  = 6
)
