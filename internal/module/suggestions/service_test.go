package suggestions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/places"
	"rantau-go/internal/module/suggestions"
	"rantau-go/internal/module/suggestions/mock"
	"rantau-go/internal/pkg/postgis"
)

// noPlaces returns a place service that fails the test if it is used.
func noPlaces(t *testing.T) *mock.MockPlaceService {
	t.Helper()
	return mock.NewMockPlaceService(gomock.NewController(t))
}

// newMocks returns a repository and place service that fail the test on any
// call they do not expect.
func newMocks(t *testing.T) (*mock.MockRepository, *mock.MockPlaceService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mock.NewMockRepository(ctrl), mock.NewMockPlaceService(ctrl)
}

// assertJSON fails unless got is the JSON document want. Numbers are compared
// by their literal text, so 1.50 and 1.5 are different.
func assertJSON(t *testing.T, got json.RawMessage, want string) {
	t.Helper()
	assert.Equal(t, decodeExact(t, json.RawMessage(want)), decodeExact(t, got), "got JSON %s, want %s", got, want)
}

func decodeExact(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	require.NoError(t, dec.Decode(&v), "invalid JSON %s", raw)
	return v
}

// submitted is what user 7 submits with only the required fields set: a warung
// in sub-district 1, category 2, pinned in Jakarta.
func submitted() *suggestions.Suggestion {
	return &suggestions.Suggestion{
		UserID:        7,
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Location:      postgis.MustPoint(106.8, -6.2),
	}
}

// createSuggestion runs Service.Create for s. It returns the suggestion the
// repository was asked to store, or nil if storage was never reached.
func createSuggestion(t *testing.T, s *suggestions.Suggestion) (*suggestions.Suggestion, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))

	var stored *suggestions.Suggestion
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *suggestions.Suggestion) (*suggestions.Suggestion, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := suggestions.NewService(repo, noPlaces(t)).Create(context.Background(), s)
	return stored, err
}

// -----------------------------------------------------------------------------
// Pending limit
// -----------------------------------------------------------------------------

func TestService_Create_MaxPendingPerUser(t *testing.T) {
	assert.Equal(t, 5, suggestions.MaxPendingPerUser)
}

// -----------------------------------------------------------------------------
// Submitting suggestions
// -----------------------------------------------------------------------------

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
		{name: "empty", input: "", wantName: "", wantErr: suggestions.ErrInvalidName},
		{name: "spaces only", input: "   ", wantName: "", wantErr: suggestions.ErrInvalidName},
		{name: "tabs and newlines only", input: "\t\n", wantName: "", wantErr: suggestions.ErrInvalidName},
		{name: "151 characters is too long", input: long + "a", wantName: "", wantErr: suggestions.ErrNameTooLong},
		{name: "151 multibyte characters is too long", input: strings.Repeat("é", 151), wantName: "", wantErr: suggestions.ErrNameTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Name = tc.input

			stored, err := createSuggestion(t, s)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "invalid name reached storage")
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
				require.NoError(t, err, "invalid test point")
				s.Location = point
			}

			stored, err := createSuggestion(t, s)
			if tc.wantErr {
				require.ErrorIs(t, err, suggestions.ErrInvalidLocation)
				require.Nil(t, stored, "missing or outside location reached storage")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored.Location, "stored location is missing")
			assert.Equal(t, tc.lon, stored.Location.X())
			assert.Equal(t, tc.lat, stored.Location.Y())
		})
	}
}

func TestService_Create_OptionalText(t *testing.T) {
	fields := []struct {
		name string
		set  func(*suggestions.Suggestion, *string)
		get  func(*suggestions.Suggestion) *string
	}{
		{name: "address",
			set: func(s *suggestions.Suggestion, v *string) { s.Address = v },
			get: func(s *suggestions.Suggestion) *string { return s.Address }},
		{name: "operating hours",
			set: func(s *suggestions.Suggestion, v *string) { s.OperatingHours = v },
			get: func(s *suggestions.Suggestion) *string { return s.OperatingHours }},
		{name: "estimated price range",
			set: func(s *suggestions.Suggestion, v *string) { s.EstimatedPriceRange = v },
			get: func(s *suggestions.Suggestion) *string { return s.EstimatedPriceRange }},
		{name: "short description",
			set: func(s *suggestions.Suggestion, v *string) { s.ShortDescription = v },
			get: func(s *suggestions.Suggestion) *string { return s.ShortDescription }},
		{name: "note",
			set: func(s *suggestions.Suggestion, v *string) { s.Note = v },
			get: func(s *suggestions.Suggestion) *string { return s.Note }},
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

				stored, err := createSuggestion(t, s)
				require.NoError(t, err)
				got := f.get(stored)
				if tc.want == nil {
					assert.Nil(t, got, "stored %s, want none", f.name)
					return
				}
				require.NotNil(t, got, "stored no %s, want %q", f.name, *tc.want)
				assert.Equal(t, *tc.want, *got, "stored %s", f.name)
			})
		}
	}
}

func TestService_Create_OptionalTextLimits(t *testing.T) {
	fields := []struct {
		name    string
		limit   int
		tooLong error
		set     func(*suggestions.Suggestion, *string)
	}{
		{name: "address", limit: places.MaxAddressLength, tooLong: suggestions.ErrAddressTooLong,
			set: func(s *suggestions.Suggestion, v *string) { s.Address = v }},
		{name: "operating hours", limit: places.MaxOperatingHoursLength, tooLong: suggestions.ErrOperatingHoursTooLong,
			set: func(s *suggestions.Suggestion, v *string) { s.OperatingHours = v }},
		{name: "estimated price range", limit: places.MaxEstimatedPriceRangeLength, tooLong: suggestions.ErrEstimatedPriceRangeTooLong,
			set: func(s *suggestions.Suggestion, v *string) { s.EstimatedPriceRange = v }},
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

				stored, err := createSuggestion(t, s)
				require.ErrorIs(t, err, tc.wantErr)
				if tc.wantErr != nil {
					require.Nil(t, stored, "refused suggestion reached storage")
				}
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
		{name: "1001 characters", note: thousand + "a", wantErr: suggestions.ErrNoteTooLong},
		{name: "1001 multibyte characters", note: strings.Repeat("é", 1001), wantErr: suggestions.ErrNoteTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Note = new(tc.note)

			stored, err := createSuggestion(t, s)
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr != nil {
				require.Nil(t, stored, "too long note reached storage")
			}
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

			stored, err := createSuggestion(t, s)
			if tc.wantErr {
				require.ErrorIs(t, err, suggestions.ErrInvalidAttributes)
				require.Nil(t, stored, "rejected attributes reached storage")
				return
			}
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantStored)
		})
	}
}

func TestService_Create_StartsPending(t *testing.T) {
	// A user cannot submit a suggestion that is already reviewed.
	s := submitted()
	s.Status = suggestions.StatusApproved
	s.RejectionReason = new("Tutup")
	s.PlaceID = new(int64(42))
	s.ReviewedBy = new(int64(99))
	s.ReviewedAt = new(time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC))

	stored, err := createSuggestion(t, s)
	require.NoError(t, err)
	assert.Equal(t, suggestions.StatusPending, stored.Status)
	assert.Nil(t, stored.RejectionReason, "review fields reached storage")
	assert.Nil(t, stored.PlaceID, "review fields reached storage")
	assert.Nil(t, stored.ReviewedBy, "review fields reached storage")
	assert.Nil(t, stored.ReviewedAt, "review fields reached storage")
}

func TestService_Create_KeepsAuthorAndCatalog(t *testing.T) {
	stored, err := createSuggestion(t, submitted())
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
		{name: "storage reports too many pending suggestions", storeErr: suggestions.ErrTooManyPending, wantErr: suggestions.ErrTooManyPending},
		{name: "storage reports an unknown sub-district", storeErr: suggestions.ErrSubDistrictNotFound, wantErr: suggestions.ErrSubDistrictNotFound},
		{name: "storage reports an unknown category", storeErr: suggestions.ErrCategoryNotFound, wantErr: suggestions.ErrCategoryNotFound},
		{name: "storage failure is returned", storeErr: boom, wantErr: boom},
		{name: "stored suggestion is returned", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&suggestions.Suggestion{ID: 3}, tc.storeErr)

			got, err := suggestions.NewService(repo, noPlaces(t)).Create(context.Background(), submitted())
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored suggestion was not returned")
				assert.Equal(t, int64(3), got.ID)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Reading suggestions
// -----------------------------------------------------------------------------

func TestService_GetSuggestion(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "returns the stored suggestion", repoErr: nil},
		{name: "unknown suggestion", repoErr: suggestions.ErrSuggestionNotFound},
		{name: "storage failure is returned", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), tc.repoErr)

			got, err := suggestions.NewService(repo, noPlaces(t)).GetSuggestion(context.Background(), 3)
			require.ErrorIs(t, err, tc.repoErr)
			if tc.repoErr == nil {
				require.NotNil(t, got, "stored suggestion was not returned")
				assert.Equal(t, int64(3), got.ID)
			}
		})
	}
}

func TestService_GetByUser(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name        string
		suggestions []suggestions.Suggestion // returned by storage
		repoErr     error                    // returned by storage
		wantIDs     []int64
	}{
		{name: "suggestions keep the stored order", suggestions: []suggestions.Suggestion{{ID: 5}, {ID: 2}, {ID: 9}}, repoErr: nil, wantIDs: []int64{5, 2, 9}},
		{name: "no suggestions", suggestions: []suggestions.Suggestion{}, repoErr: nil, wantIDs: []int64{}},
		{name: "storage failure", suggestions: nil, repoErr: boom, wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetByUser(gomock.Any(), int64(7)).Return(tc.suggestions, tc.repoErr)

			got, err := suggestions.NewService(repo, noPlaces(t)).GetByUser(context.Background(), 7)
			require.ErrorIs(t, err, tc.repoErr)
			if tc.repoErr != nil {
				return
			}

			ids := make([]int64, 0, len(got))
			for _, s := range got {
				ids = append(ids, s.ID)
			}
			assert.Equal(t, tc.wantIDs, ids)
		})
	}
}

func TestService_GetMany_Status(t *testing.T) {
	tests := []struct {
		name    string
		status  suggestions.Status
		wantErr error
	}{
		{name: "no status lists every suggestion", status: "", wantErr: nil},
		{name: "pending", status: suggestions.StatusPending, wantErr: nil},
		{name: "approved", status: suggestions.StatusApproved, wantErr: nil},
		{name: "rejected", status: suggestions.StatusRejected, wantErr: nil},
		{name: "unknown status", status: "deleted", wantErr: suggestions.ErrInvalidStatus},
		{name: "status is case sensitive", status: "PENDING", wantErr: suggestions.ErrInvalidStatus},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filter := suggestions.SuggestionFilter{Status: tc.status}

			repo := mock.NewMockRepository(gomock.NewController(t))
			if tc.wantErr == nil {
				repo.EXPECT().GetMany(gomock.Any(), filter).Return([]suggestions.Suggestion{}, nil)
			}

			_, err := suggestions.NewService(repo, noPlaces(t)).GetMany(context.Background(), filter)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_GetMany_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, boom)

	_, err := suggestions.NewService(repo, noPlaces(t)).GetMany(context.Background(),
		suggestions.SuggestionFilter{Status: suggestions.StatusPending})
	require.ErrorIs(t, err, boom)
}

// -----------------------------------------------------------------------------
// Approving suggestions
// -----------------------------------------------------------------------------

const adminID int64 = 99

// pending is suggestion 3, stored and waiting for review.
func pending() *suggestions.Suggestion {
	s := submitted()
	s.ID = 3
	s.Status = suggestions.StatusPending
	s.Attributes = json.RawMessage(`{}`)
	s.CreatedAt = time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC)
	return s
}

// approved is suggestion 3 after admin 99 approved it as place 42.
func approved() *suggestions.Suggestion {
	s := pending()
	s.Status = suggestions.StatusApproved
	s.PlaceID = new(int64(42))
	s.ReviewedBy = new(adminID)
	s.ReviewedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

// rejected is suggestion 3 after admin 99 rejected it.
func rejected() *suggestions.Suggestion {
	s := pending()
	s.Status = suggestions.StatusRejected
	s.RejectionReason = new("Sudah tutup")
	s.ReviewedBy = new(adminID)
	s.ReviewedAt = new(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

// finalPlace is the place the admin approves suggestion 3 as, after
// correcting it.
func finalPlace() *places.Place {
	return &places.Place{
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Address:       new("Jl. Sabang 12"),
		Location:      postgis.MustPoint(106.8, -6.2),
		Attributes:    json.RawMessage(`{"halal":true}`),
	}
}

func TestService_Approve(t *testing.T) {
	repo, placeSvc := newMocks(t)
	want := approved()

	gomock.InOrder(
		repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), nil),
		placeSvc.EXPECT().Create(gomock.Any(), finalPlace()).Return(&places.Place{ID: 42}, nil),
		repo.EXPECT().MarkApproved(gomock.Any(), int64(3), int64(42), adminID).Return(want, nil),
	)

	got, err := suggestions.NewService(repo, placeSvc).Approve(context.Background(), 3, adminID, finalPlace())
	require.NoError(t, err)
	assert.Same(t, want, got, "want the approved suggestion")
}

func TestService_Approve_PlaceWithoutLocation(t *testing.T) {
	// Location is required for suggestions, so the approved place must have
	// one too. Nothing may be read, created, or marked.
	repo, placeSvc := newMocks(t)
	p := finalPlace()
	p.Location = nil

	_, err := suggestions.NewService(repo, placeSvc).Approve(context.Background(), 3, adminID, p)
	require.ErrorIs(t, err, suggestions.ErrInvalidLocation)
}

func TestService_Approve_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name   string
		getErr error
	}{
		{name: "unknown suggestion", getErr: suggestions.ErrSuggestionNotFound},
		{name: "lookup failure is returned", getErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No place may be created for a suggestion that cannot be read.
			repo, placeSvc := newMocks(t)
			repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(nil, tc.getErr)

			_, err := suggestions.NewService(repo, placeSvc).Approve(context.Background(), 3, adminID, finalPlace())
			require.ErrorIs(t, err, tc.getErr)
		})
	}
}

func TestService_Approve_AlreadyReviewed(t *testing.T) {
	tests := []struct {
		name     string
		existing *suggestions.Suggestion
	}{
		{name: "already approved", existing: approved()},
		{name: "already rejected", existing: rejected()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No place may be created for a reviewed suggestion.
			repo, placeSvc := newMocks(t)
			repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(tc.existing, nil)

			_, err := suggestions.NewService(repo, placeSvc).Approve(context.Background(), 3, adminID, finalPlace())
			require.ErrorIs(t, err, suggestions.ErrAlreadyReviewed)
		})
	}
}

func TestService_Approve_PlaceRejected(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		createErr error
	}{
		{name: "invalid name", createErr: places.ErrInvalidName},
		{name: "invalid attributes", createErr: places.ErrInvalidAttributes},
		{name: "unknown category", createErr: places.ErrCategoryNotFound},
		{name: "unknown sub-district", createErr: places.ErrSubDistrictNotFound},
		{name: "place storage failure", createErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The suggestion stays pending: it may not be marked approved.
			repo, placeSvc := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), nil),
				placeSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.createErr),
			)

			_, err := suggestions.NewService(repo, placeSvc).Approve(context.Background(), 3, adminID, finalPlace())
			require.ErrorIs(t, err, tc.createErr)
		})
	}
}

func TestService_Approve_MarkFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		markErr   error
		deleteErr error
	}{
		{name: "another admin reviewed it meanwhile", markErr: suggestions.ErrAlreadyReviewed, deleteErr: nil},
		{name: "suggestion deleted meanwhile", markErr: suggestions.ErrSuggestionNotFound, deleteErr: nil},
		{name: "storage failure is returned", markErr: boom, deleteErr: nil},
		{name: "created place delete failure is ignored", markErr: suggestions.ErrAlreadyReviewed, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The created place is removed, so no place exists without an
			// approved suggestion.
			repo, placeSvc := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), nil),
				placeSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&places.Place{ID: 42}, nil),
				repo.EXPECT().MarkApproved(gomock.Any(), int64(3), int64(42), adminID).Return(nil, tc.markErr),
				placeSvc.EXPECT().Delete(gomock.Any(), int64(42)).Return(tc.deleteErr),
			)

			_, err := suggestions.NewService(repo, placeSvc).Approve(context.Background(), 3, adminID, finalPlace())
			require.ErrorIs(t, err, tc.markErr)
		})
	}
}

// -----------------------------------------------------------------------------
// Rejecting suggestions
// -----------------------------------------------------------------------------

func TestService_Reject_Reason(t *testing.T) {
	thousand := strings.Repeat("a", 1000)

	tests := []struct {
		name   string
		reason *string
		want   *string
	}{
		{name: "absent reason stays absent", reason: nil, want: nil},
		{name: "surrounding spaces are removed", reason: new("  Sudah tutup  "), want: new("Sudah tutup")},
		{name: "inner line breaks are kept", reason: new("Sudah tutup.\nPindah."), want: new("Sudah tutup.\nPindah.")},
		{name: "empty reason is stored as absent", reason: new(""), want: nil},
		{name: "blank reason is stored as absent", reason: new(" \t\n "), want: nil},
		{name: "exactly 1000 characters", reason: new(thousand), want: new(thousand)},
		{name: "1000 multibyte characters", reason: new(strings.Repeat("é", 1000)), want: new(strings.Repeat("é", 1000))},
		{name: "surrounding spaces do not count toward the limit", reason: new(" " + thousand + " "), want: new(thousand)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))

			var stored *string
			called := false
			repo.EXPECT().Reject(gomock.Any(), int64(3), adminID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _, _ int64, reason *string) (*suggestions.Suggestion, error) {
					called, stored = true, reason
					return rejected(), nil
				})

			_, err := suggestions.NewService(repo, noPlaces(t)).Reject(context.Background(), 3, adminID, tc.reason)
			require.NoError(t, err)
			require.True(t, called, "rejection was not stored")
			if tc.want == nil {
				assert.Nil(t, stored, "stored reason, want none")
				return
			}
			require.NotNil(t, stored, "stored no reason, want %q", *tc.want)
			assert.Equal(t, *tc.want, *stored)
		})
	}
}

func TestService_Reject_ReasonTooLong(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{name: "1001 characters", reason: strings.Repeat("a", 1001)},
		{name: "1001 multibyte characters", reason: strings.Repeat("é", 1001)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a too long reason must never reach storage.
			repo := mock.NewMockRepository(gomock.NewController(t))

			_, err := suggestions.NewService(repo, noPlaces(t)).Reject(context.Background(), 3, adminID, new(tc.reason))
			require.ErrorIs(t, err, suggestions.ErrReasonTooLong)
		})
	}
}

func TestService_Reject_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		rejectErr error
	}{
		{name: "unknown suggestion", rejectErr: suggestions.ErrSuggestionNotFound},
		{name: "already reviewed", rejectErr: suggestions.ErrAlreadyReviewed},
		{name: "storage failure is returned", rejectErr: boom},
		{name: "rejected suggestion is returned", rejectErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := rejected()

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Reject(gomock.Any(), int64(3), adminID, gomock.Any()).Return(want, tc.rejectErr)

			got, err := suggestions.NewService(repo, noPlaces(t)).Reject(context.Background(), 3, adminID, new("Sudah tutup"))
			require.ErrorIs(t, err, tc.rejectErr)
			if tc.rejectErr == nil {
				assert.Same(t, want, got, "want the rejected suggestion")
			}
		})
	}
}
