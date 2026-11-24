package suggestions_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/suggestions"
	"sakukota-go/internal/module/suggestions/mock"
	"sakukota-go/internal/pkg/postgis"
)

// txKey marks the context that newService's transactor passes to the
// transaction's function.
type txKey struct{}

// inTx matches a context carrying the transaction newService's transactor started.
var inTx = gomock.Cond(func(ctx context.Context) bool { return ctx.Value(txKey{}) != nil })

// newService returns a Service over mock collaborators. Its transactor runs
// each function it is given with a context marked by txKey, returning the
// function's error.
func newService(t *testing.T) (suggestions.Service, *mock.MockRepository, *mock.MockPlaceService) {
	t.Helper()

	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	places := mock.NewMockPlaceService(ctrl)
	tx := mock.NewMockTransactor(ctrl)
	tx.EXPECT().WithinTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(context.WithValue(ctx, txKey{}, true))
	}).AnyTimes()

	return suggestions.NewService(repo, tx, places), repo, places
}

// submitted is what user 7 submits with only the required fields set: a warung
// in sub-district 1, category 2, pinned in Jakarta.
func submitted() *domain.Suggestion {
	return &domain.Suggestion{
		UserID:        7,
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Location:      postgis.MustPoint(106.8, -6.2),
	}
}

// createSuggestion runs Service.Create for s. If reachesStorage, the
// repository accepts the suggestion and it is returned as stored; otherwise
// the repository must not be called and the returned suggestion is nil.
func createSuggestion(t *testing.T, s *domain.Suggestion, reachesStorage bool) (*domain.Suggestion, error) {
	t.Helper()

	svc, repo, _ := newService(t)

	var stored *domain.Suggestion
	if reachesStorage {
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *domain.Suggestion) (*domain.Suggestion, error) {
			stored = got
			return got, nil
		})
	}

	_, err := svc.Create(t.Context(), s)
	return stored, err
}

func TestService_Create_MaxPendingPerUser(t *testing.T) {
	assert.Equal(t, 5, suggestions.MaxPendingPerUser)
}

func TestService_Create_Name(t *testing.T) {
	long := strings.Repeat("a", 150)

	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  error
	}{
		{name: "surrounding spaces are removed", input: "  Warung Bu Ani  ", wantName: "Warung Bu Ani", wantErr: nil},
		{name: "tabs and newlines are removed", input: "\tWarung Bu Ani\n", wantName: "Warung Bu Ani", wantErr: nil},
		{name: "inner spacing is kept", input: "Warung  Bu Ani", wantName: "Warung  Bu Ani", wantErr: nil},
		{name: "150 characters is allowed", input: long, wantName: long, wantErr: nil},
		{name: "150 multibyte characters is allowed", input: strings.Repeat("é", 150), wantName: strings.Repeat("é", 150), wantErr: nil},
		{name: "length is counted after trimming", input: "  " + long + "  ", wantName: long, wantErr: nil},
		{name: "empty", input: "", wantName: "", wantErr: domain.ErrPlaceNameRequired},
		{name: "spaces only", input: "   ", wantName: "", wantErr: domain.ErrPlaceNameRequired},
		{name: "tabs and newlines only", input: "\t\n", wantName: "", wantErr: domain.ErrPlaceNameRequired},
		{name: "151 characters is too long", input: long + "a", wantName: "", wantErr: domain.ErrPlaceNameTooLong},
		{name: "151 multibyte characters is too long", input: strings.Repeat("é", 151), wantName: "", wantErr: domain.ErrPlaceNameTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Name = tc.input

			stored, err := createSuggestion(t, s, tc.wantErr == nil)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, stored.Name)
		})
	}
}

func TestService_Create_Location(t *testing.T) {
	tests := []struct {
		name     string
		lon, lat float64
		omit     bool // no location in the suggestion
		wantErr  bool
	}{
		{name: "Jakarta", lon: 106.8, lat: -6.2, omit: false, wantErr: false},
		{name: "Sabang, the western tip", lon: 95.3, lat: 5.9, omit: false, wantErr: false},
		{name: "Merauke, the eastern tip", lon: 140.4, lat: -8.5, omit: false, wantErr: false},
		{name: "west edge of the box", lon: 95, lat: -6.2, omit: false, wantErr: false},
		{name: "east edge of the box", lon: 141, lat: -6.2, omit: false, wantErr: false},
		{name: "north edge of the box", lon: 106.8, lat: 6, omit: false, wantErr: false},
		{name: "south edge of the box", lon: 106.8, lat: -11, omit: false, wantErr: false},
		{name: "location omitted", lon: 0, lat: 0, omit: true, wantErr: true},
		{name: "Tokyo", lon: 139.7, lat: 35.7, omit: false, wantErr: true},
		{name: "null island", lon: 0, lat: 0, omit: false, wantErr: true},
		{name: "just west of the box", lon: 94.99, lat: -6.2, omit: false, wantErr: true},
		{name: "just east of the box", lon: 141.01, lat: -6.2, omit: false, wantErr: true},
		{name: "just north of the box", lon: 106.8, lat: 6.01, omit: false, wantErr: true},
		{name: "just south of the box", lon: 106.8, lat: -11.01, omit: false, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Location = nil
			if !tc.omit {
				point, err := postgis.NewPoint(tc.lon, tc.lat)
				require.NoError(t, err)
				s.Location = point
			}

			stored, err := createSuggestion(t, s, !tc.wantErr)
			if tc.wantErr {
				assert.ErrorIs(t, err, domain.ErrSuggestionLocationInvalid)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored.Location)
			assert.Equal(t, tc.lon, stored.Location.X())
			assert.Equal(t, tc.lat, stored.Location.Y())
		})
	}
}

func TestService_Create_OptionalText(t *testing.T) {
	fields := []struct {
		name string
		set  func(*domain.Suggestion, *string)
		get  func(*domain.Suggestion) *string
	}{
		{name: "address",
			set: func(s *domain.Suggestion, v *string) { s.Address = v },
			get: func(s *domain.Suggestion) *string { return s.Address }},
		{name: "operating_hours",
			set: func(s *domain.Suggestion, v *string) { s.OperatingHours = v },
			get: func(s *domain.Suggestion) *string { return s.OperatingHours }},
		{name: "estimated_price_range",
			set: func(s *domain.Suggestion, v *string) { s.EstimatedPriceRange = v },
			get: func(s *domain.Suggestion) *string { return s.EstimatedPriceRange }},
		{name: "short_description",
			set: func(s *domain.Suggestion, v *string) { s.ShortDescription = v },
			get: func(s *domain.Suggestion) *string { return s.ShortDescription }},
		{name: "note",
			set: func(s *domain.Suggestion, v *string) { s.Note = v },
			get: func(s *domain.Suggestion) *string { return s.Note }},
	}

	tests := []struct {
		name  string
		input *string
		want  *string
	}{
		{name: "absent stays absent", input: nil, want: nil},
		{name: "surrounding spaces are removed", input: new("  Jl. Sabang 12  "), want: new("Jl. Sabang 12")},
		{name: "tabs and newlines are removed", input: new("\tJl. Sabang 12\n"), want: new("Jl. Sabang 12")},
		{name: "inner line breaks are kept", input: new("Murah.\nPorsi besar."), want: new("Murah.\nPorsi besar.")},
		{name: "empty is stored as absent", input: new(""), want: nil},
		{name: "blank is stored as absent", input: new(" \t\n "), want: nil},
	}

	for _, f := range fields {
		for _, tc := range tests {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				s := submitted()
				f.set(s, tc.input)

				stored, err := createSuggestion(t, s, true)
				require.NoError(t, err)
				assert.Equal(t, tc.want, f.get(stored))
			})
		}
	}
}

func TestService_Create_OptionalTextLimits(t *testing.T) {
	fields := []struct {
		name    string
		limit   int
		tooLong error
		set     func(*domain.Suggestion, *string)
	}{
		{name: "address", limit: domain.MaxPlaceAddressLength, tooLong: domain.ErrPlaceAddressTooLong,
			set: func(s *domain.Suggestion, v *string) { s.Address = v }},
		{name: "operating_hours", limit: domain.MaxPlaceOperatingHoursLength, tooLong: domain.ErrPlaceOperatingHoursTooLong,
			set: func(s *domain.Suggestion, v *string) { s.OperatingHours = v }},
		{name: "estimated_price_range", limit: domain.MaxPlaceEstimatedPriceRangeLength, tooLong: domain.ErrPlaceEstimatedPriceRangeTooLong,
			set: func(s *domain.Suggestion, v *string) { s.EstimatedPriceRange = v }},
	}

	for _, f := range fields {
		tests := []struct {
			name    string
			input   string
			wantErr error // nil means the text is accepted
		}{
			{name: "at the limit", input: strings.Repeat("a", f.limit)},
			{name: "multibyte characters count once each", input: strings.Repeat("é", f.limit)},
			{name: "length is counted after trimming", input: "  " + strings.Repeat("a", f.limit) + "  "},
			{name: "one character over the limit", input: strings.Repeat("a", f.limit+1), wantErr: f.tooLong},
		}

		for _, tc := range tests {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				s := submitted()
				input := tc.input
				f.set(s, &input)

				_, err := createSuggestion(t, s, tc.wantErr == nil)
				if tc.wantErr != nil {
					assert.ErrorIs(t, err, tc.wantErr)
					return
				}
				assert.NoError(t, err)
			})
		}
	}
}

func TestService_Create_NoteLength(t *testing.T) {
	thousand := strings.Repeat("a", 1000)

	tests := []struct {
		name    string
		note    string
		wantErr error
	}{
		{name: "exactly 1000 characters", note: thousand, wantErr: nil},
		{name: "1000 multibyte characters", note: strings.Repeat("é", 1000), wantErr: nil},
		{name: "surrounding spaces do not count toward the limit", note: " " + thousand + " ", wantErr: nil},
		{name: "1001 characters", note: thousand + "a", wantErr: domain.ErrSuggestionNoteTooLong},
		{name: "1001 multibyte characters", note: strings.Repeat("é", 1001), wantErr: domain.ErrSuggestionNoteTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Note = new(tc.note)

			_, err := createSuggestion(t, s, tc.wantErr == nil)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestService_Create_Attributes(t *testing.T) {
	// Attributes are only checked to be an object here. They are checked
	// against the category's fields when an admin approves the suggestion.
	tests := []struct {
		name       string
		attributes string
		wantStored string // stored JSON when the attributes are accepted
		wantErr    bool
	}{
		{name: "absent attributes are stored as an empty object", attributes: "", wantStored: `{}`, wantErr: false},
		{name: "null attributes are stored as an empty object", attributes: `null`, wantStored: `{}`, wantErr: false},
		{name: "empty object is kept", attributes: `{}`, wantStored: `{}`, wantErr: false},
		{name: "object is kept verbatim", attributes: `{"halal":true,"seats":1.50}`, wantStored: `{"halal":true,"seats":1.50}`, wantErr: false},
		{name: "keys unknown to the category are kept", attributes: `{"parking":true}`, wantStored: `{"parking":true}`, wantErr: false},
		{name: "array instead of object", attributes: `[1,2]`, wantStored: "", wantErr: true},
		{name: "string instead of object", attributes: `"text"`, wantStored: "", wantErr: true},
		{name: "number instead of object", attributes: `5`, wantStored: "", wantErr: true},
		{name: "malformed JSON", attributes: `{"halal":`, wantStored: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Attributes = json.RawMessage(tc.attributes)

			stored, err := createSuggestion(t, s, !tc.wantErr)
			if tc.wantErr {
				assert.ErrorIs(t, err, domain.ErrSuggestionAttributesInvalid)
				return
			}
			require.NoError(t, err)
			// Compared as text, so 1.50 and 1.5 are different.
			assert.Equal(t, tc.wantStored, string(stored.Attributes))
		})
	}
}

func TestService_Create_KeepsAuthorAndCatalog(t *testing.T) {
	stored, err := createSuggestion(t, submitted(), true)
	require.NoError(t, err)
	assert.Equal(t, int64(7), stored.UserID)
	assert.Equal(t, int64(1), stored.SubDistrictID)
	assert.Equal(t, int64(2), stored.CategoryID)
}

func TestService_Create_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name     string
		storeErr error // returned when the suggestion is stored
		wantErr  error // nil means the call succeeds
	}{
		{name: "too_many_pending", storeErr: domain.ErrSuggestionLimitReached, wantErr: domain.ErrSuggestionLimitReached},
		{name: "unknown_sub_district", storeErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "unknown_category", storeErr: domain.ErrCategoryNotFound, wantErr: domain.ErrCategoryNotFound},
		{name: "storage_failure", storeErr: boom, wantErr: boom},
		{name: "stored_returned", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&domain.Suggestion{ID: 3}, tc.storeErr)

			got, err := svc.Create(t.Context(), submitted())
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, int64(3), got.ID)
		})
	}
}

func TestService_Fetch(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "returns_stored", repoErr: nil},
		{name: "unknown", repoErr: domain.ErrSuggestionNotFound},
		{name: "storage_failure", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(pending(), tc.repoErr)

			got, err := svc.Fetch(t.Context(), 3)
			if tc.repoErr != nil {
				assert.ErrorIs(t, err, tc.repoErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, int64(3), got.ID)
		})
	}
}

func TestService_ListByUser(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name        string
		suggestions []domain.Suggestion // returned by storage
		repoErr     error               // returned by storage
		wantIDs     []int64
	}{
		{name: "keeps_stored_order", suggestions: []domain.Suggestion{{ID: 5}, {ID: 2}, {ID: 9}}, repoErr: nil, wantIDs: []int64{5, 2, 9}},
		{name: "none", suggestions: []domain.Suggestion{}, repoErr: nil, wantIDs: []int64{}},
		{name: "storage_failure", suggestions: nil, repoErr: boom, wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			repo.EXPECT().ListByUser(gomock.Any(), int64(7)).Return(tc.suggestions, tc.repoErr)

			got, err := svc.ListByUser(t.Context(), 7)
			if tc.repoErr != nil {
				assert.ErrorIs(t, err, tc.repoErr)
				return
			}
			require.NoError(t, err)

			ids := make([]int64, 0, len(got))
			for _, s := range got {
				ids = append(ids, s.ID)
			}
			assert.Equal(t, tc.wantIDs, ids)
		})
	}
}

func TestService_List_Status(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.Status
		wantErr error
	}{
		{name: "no_status", status: "", wantErr: nil},
		{name: "pending", status: domain.StatusPending, wantErr: nil},
		{name: "approved", status: domain.StatusApproved, wantErr: nil},
		{name: "rejected", status: domain.StatusRejected, wantErr: nil},
		{name: "unknown", status: "deleted", wantErr: domain.ErrSuggestionStatusInvalid},
		{name: "case_sensitive", status: "PENDING", wantErr: domain.ErrSuggestionStatusInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			filter := suggestions.Filter{Status: tc.status}
			if tc.wantErr == nil {
				repo.EXPECT().List(gomock.Any(), filter).Return([]domain.Suggestion{}, nil)
			}

			_, err := svc.List(t.Context(), filter)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestService_List_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	svc, repo, _ := newService(t)
	filter := suggestions.Filter{Status: domain.StatusPending}
	repo.EXPECT().List(gomock.Any(), filter).Return(nil, boom)

	_, err := svc.List(t.Context(), filter)
	assert.ErrorIs(t, err, boom)
}

const adminID int64 = 99

// pending is suggestion 3, stored and waiting for review.
func pending() *domain.Suggestion {
	s := submitted()
	s.ID = 3
	s.Status = domain.StatusPending
	s.Attributes = json.RawMessage(`{}`)
	s.CreatedAt = time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC)
	return s
}

// approved is suggestion 3 after admin 99 approved it as place 42.
func approved() *domain.Suggestion {
	s := pending()
	s.Status = domain.StatusApproved
	s.PlaceID = new(int64(42))
	s.ReviewedBy = new(adminID)
	s.ReviewedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

// rejected is suggestion 3 after admin 99 rejected it.
func rejected() *domain.Suggestion {
	s := pending()
	s.Status = domain.StatusRejected
	s.RejectionReason = new("Sudah tutup")
	s.ReviewedBy = new(adminID)
	s.ReviewedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

// finalPlace is the place the admin approves suggestion 3 as, after
// correcting it.
func finalPlace() *domain.Place {
	return &domain.Place{
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Address:       new("Jl. Sabang 12"),
		Location:      postgis.MustPoint(106.8, -6.2),
		Attributes:    json.RawMessage(`{"halal":true}`),
	}
}

func TestService_Approve(t *testing.T) {
	want := approved()

	svc, repo, places := newService(t)
	gomock.InOrder(
		repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(pending(), nil),
		// The place and the approval are written in the one transaction.
		places.EXPECT().Create(inTx, finalPlace()).Return(&domain.Place{ID: 42}, nil),
		repo.EXPECT().MarkApproved(inTx, int64(3), int64(42), adminID).Return(want, nil),
	)

	got, err := svc.Approve(t.Context(), 3, adminID, finalPlace())
	require.NoError(t, err)
	assert.Same(t, want, got)
}

func TestService_Approve_PlaceWithoutLocation(t *testing.T) {
	// Location is required for suggestions, so the approved place must have
	// one too. Nothing may be read, created, or marked.
	svc, _, _ := newService(t)
	p := finalPlace()
	p.Location = nil

	_, err := svc.Approve(t.Context(), 3, adminID, p)
	assert.ErrorIs(t, err, domain.ErrSuggestionLocationInvalid)
}

func TestService_Approve_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name   string
		getErr error
	}{
		{name: "unknown", getErr: domain.ErrSuggestionNotFound},
		{name: "lookup_failure", getErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No place may be created for a suggestion that cannot be read.
			svc, repo, _ := newService(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(nil, tc.getErr)

			_, err := svc.Approve(t.Context(), 3, adminID, finalPlace())
			assert.ErrorIs(t, err, tc.getErr)
		})
	}
}

func TestService_Approve_AlreadyReviewed(t *testing.T) {
	tests := []struct {
		name     string
		existing *domain.Suggestion
	}{
		{name: "approved", existing: approved()},
		{name: "rejected", existing: rejected()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No place may be created for a reviewed suggestion.
			svc, repo, _ := newService(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(tc.existing, nil)

			_, err := svc.Approve(t.Context(), 3, adminID, finalPlace())
			assert.ErrorIs(t, err, domain.ErrSuggestionAlreadyReviewed)
		})
	}
}

func TestService_Approve_PlaceRejected(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		createErr error
	}{
		{name: "invalid_name", createErr: domain.ErrPlaceNameRequired},
		{name: "invalid_attributes", createErr: domain.ErrPlaceAttributesInvalid},
		{name: "unknown_category", createErr: domain.ErrCategoryNotFound},
		{name: "unknown_sub_district", createErr: domain.ErrSubDistrictNotFound},
		{name: "place_storage_failure", createErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The suggestion stays pending: it may not be marked approved.
			svc, repo, places := newService(t)
			gomock.InOrder(
				repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(pending(), nil),
				places.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.createErr),
			)

			_, err := svc.Approve(t.Context(), 3, adminID, finalPlace())
			assert.ErrorIs(t, err, tc.createErr)
		})
	}
}

func TestService_Approve_MarkFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		markErr error
	}{
		{name: "reviewed_meanwhile", markErr: domain.ErrSuggestionAlreadyReviewed},
		{name: "deleted_meanwhile", markErr: domain.ErrSuggestionNotFound},
		{name: "storage_failure", markErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The error rolls back the transaction that created the place, so
			// no place exists without an approved suggestion.
			svc, repo, places := newService(t)
			gomock.InOrder(
				repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(pending(), nil),
				places.EXPECT().Create(inTx, gomock.Any()).Return(&domain.Place{ID: 42}, nil),
				repo.EXPECT().MarkApproved(inTx, int64(3), int64(42), adminID).Return(nil, tc.markErr),
			)

			got, err := svc.Approve(t.Context(), 3, adminID, finalPlace())
			assert.ErrorIs(t, err, tc.markErr)
			assert.Nil(t, got)
		})
	}
}

func TestService_Approve_TransactionFailure(t *testing.T) {
	boom := errors.New("connection reset")
	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(pending(), nil)
	tx := mock.NewMockTransactor(ctrl)
	tx.EXPECT().WithinTx(gomock.Any(), gomock.Any()).Return(boom)

	got, err := suggestions.NewService(repo, tx, mock.NewMockPlaceService(ctrl)).Approve(t.Context(), 3, adminID, finalPlace())

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestService_Reject_Reason(t *testing.T) {
	thousand := strings.Repeat("a", 1000)

	tests := []struct {
		name   string
		reason *string
		want   *string
	}{
		{name: "absent_stays_absent", reason: nil, want: nil},
		{name: "surrounding_spaces_removed", reason: new("  Sudah tutup  "), want: new("Sudah tutup")},
		{name: "inner_line_breaks_kept", reason: new("Sudah tutup.\nPindah."), want: new("Sudah tutup.\nPindah.")},
		{name: "empty_stored_as_absent", reason: new(""), want: nil},
		{name: "blank_stored_as_absent", reason: new(" \t\n "), want: nil},
		{name: "exactly_1000_characters", reason: new(thousand), want: new(thousand)},
		{name: "1000_multibyte_characters", reason: new(strings.Repeat("é", 1000)), want: new(strings.Repeat("é", 1000))},
		{name: "surrounding_spaces_not_counted", reason: new(" " + thousand + " "), want: new(thousand)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			repo.EXPECT().Reject(gomock.Any(), int64(3), adminID, tc.want).Return(rejected(), nil)

			_, err := svc.Reject(t.Context(), 3, adminID, tc.reason)
			assert.NoError(t, err)
		})
	}
}

func TestService_Reject_ReasonTooLong(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{name: "1001_characters", reason: strings.Repeat("a", 1001)},
		{name: "1001_multibyte_characters", reason: strings.Repeat("é", 1001)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A too long reason must never reach storage.
			svc, _, _ := newService(t)

			_, err := svc.Reject(t.Context(), 3, adminID, new(tc.reason))
			assert.ErrorIs(t, err, domain.ErrSuggestionReasonTooLong)
		})
	}
}

func TestService_Reject_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		rejectErr error
	}{
		{name: "unknown", rejectErr: domain.ErrSuggestionNotFound},
		{name: "already_reviewed", rejectErr: domain.ErrSuggestionAlreadyReviewed},
		{name: "storage_failure", rejectErr: boom},
		{name: "returns_rejected", rejectErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := rejected()

			svc, repo, _ := newService(t)
			repo.EXPECT().Reject(gomock.Any(), int64(3), adminID, new("Sudah tutup")).Return(want, tc.rejectErr)

			got, err := svc.Reject(t.Context(), 3, adminID, new("Sudah tutup"))
			if tc.rejectErr != nil {
				assert.ErrorIs(t, err, tc.rejectErr)
				return
			}
			require.NoError(t, err)
			assert.Same(t, want, got)
		})
	}
}
