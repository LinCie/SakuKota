package domain

import "time"

// ReportReason is why a user reports a place or a review.
type ReportReason string

// Reasons a report can give, matching the CHECK on REPORTS.reason.
const (
	ReportReasonSpam      ReportReason = "spam"
	ReportReasonOffensive ReportReason = "offensive"
	ReportReasonFake      ReportReason = "fake"
	ReportReasonOther     ReportReason = "other"
)

// ReportStatus is where a report stands. A report is resolved once, so it
// leaves pending for removed or dismissed and never changes again.
type ReportStatus string

// Statuses a report can have.
const (
	ReportStatusPending ReportStatus = "pending"
	// ReportStatusRemoved means an admin deleted the reported content.
	ReportStatusRemoved ReportStatus = "removed"
	// ReportStatusDismissed means an admin kept the reported content.
	ReportStatusDismissed ReportStatus = "dismissed"
)

// MaxReportNoteLength caps the characters in a report's note, counted after
// trimming spaces.
const MaxReportNoteLength = 1000

// Report represents a row in the REPORTS table, joined with its author.
type Report struct {
	ID     int64 `db:"id"`
	UserID int64 `db:"user_id"`
	// PlaceID is the reported place, or the place of the reported review.
	PlaceID int64 `db:"place_id"`
	// ReviewID is the reported review, or nil if the place itself is reported.
	ReviewID   *int64       `db:"review_id"`
	Reason     ReportReason `db:"reason"`
	Note       *string      `db:"note"`
	Status     ReportStatus `db:"status"`
	ResolvedBy *int64       `db:"resolved_by"`
	ResolvedAt *time.Time   `db:"resolved_at"`
	CreatedAt  time.Time    `db:"created_at"`
	User       Author       `db:"user"`
	// Place and Review are the reported content as it is now, loaded only
	// for one report. Place is nil once the place is deleted; Review is nil
	// for a place report or once the review is deleted.
	Place  *Place  `db:"-"`
	Review *Review `db:"-"`
}
