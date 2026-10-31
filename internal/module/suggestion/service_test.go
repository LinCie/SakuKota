package suggestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/place"
	"rantau-go/internal/module/suggestion"
	"rantau-go/internal/module/suggestion/mock"
	"rantau-go/internal/postgis"
)

func ptr[T any](v T) *T {
	return &v
}

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
	if !reflect.DeepEqual(decodeExact(t, got), decodeExact(t, json.RawMessage(want))) {
		t.Errorf("got JSON %s, want %s", got, want)
	}
}

func decodeExact(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("invalid JSON %s: %v", raw, err)
	}
	return v
}

// submitted is what user 7 submits with only the required fields set: a warung
// in sub-district 1, category 2, pinned in Jakarta.
func submitted() *suggestion.Suggestion {
	return &suggestion.Suggestion{
		UserID:        7,
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Location:      postgis.MustPoint(106.8, -6.2),
	}
}

// createSuggestion runs Service.Create for s. It returns the suggestion the
// repository was asked to store, or nil if storage was never reached.
func createSuggestion(t *testing.T, s *suggestion.Suggestion) (*suggestion.Suggestion, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))

	var stored *suggestion.Suggestion
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *suggestion.Suggestion) (*suggestion.Suggestion, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := suggestion.NewService(repo, noPlaces(t)).Create(context.Background(), s)
	return stored, err
}

// -----------------------------------------------------------------------------
// Pending limit
// -----------------------------------------------------------------------------

func TestService_Create_MaxPendingPerUser(t *testing.T) {
	if suggestion.MaxPendingPerUser != 5 {
		t.Errorf("MaxPendingPerUser = %d, want 5", suggestion.MaxPendingPerUser)
	}
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
		{name: "empty", input: "", wantName: "", wantErr: suggestion.ErrInvalidName},
		{name: "spaces only", input: "   ", wantName: "", wantErr: suggestion.ErrInvalidName},
		{name: "tabs and newlines only", input: "\t\n", wantName: "", wantErr: suggestion.ErrInvalidName},
		{name: "151 characters is too long", input: long + "a", wantName: "", wantErr: suggestion.ErrNameTooLong},
		{name: "151 multibyte characters is too long", input: strings.Repeat("é", 151), wantName: "", wantErr: suggestion.ErrNameTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Name = tc.input

			stored, err := createSuggestion(t, s)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatalf("invalid name reached storage as %q", stored.Name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored.Name != tc.wantName {
				t.Errorf("stored name = %q, want %q", stored.Name, tc.wantName)
			}
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
				if err != nil {
					t.Fatalf("invalid test point: %v", err)
				}
				s.Location = point
			}

			stored, err := createSuggestion(t, s)
			if tc.wantErr {
				if !errors.Is(err, suggestion.ErrInvalidLocation) {
					t.Fatalf("err = %v, want ErrInvalidLocation", err)
				}
				if stored != nil {
					t.Fatal("missing or outside location reached storage")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored.Location == nil || stored.Location.X() != tc.lon || stored.Location.Y() != tc.lat {
				t.Errorf("stored location = %v, want [%v, %v]", stored.Location, tc.lon, tc.lat)
			}
		})
	}
}

func TestService_Create_OptionalText(t *testing.T) {
	fields := []struct {
		name string
		set  func(*suggestion.Suggestion, *string)
		get  func(*suggestion.Suggestion) *string
	}{
		{name: "address",
			set: func(s *suggestion.Suggestion, v *string) { s.Address = v },
			get: func(s *suggestion.Suggestion) *string { return s.Address }},
		{name: "operating hours",
			set: func(s *suggestion.Suggestion, v *string) { s.OperatingHours = v },
			get: func(s *suggestion.Suggestion) *string { return s.OperatingHours }},
		{name: "estimated price range",
			set: func(s *suggestion.Suggestion, v *string) { s.EstimatedPriceRange = v },
			get: func(s *suggestion.Suggestion) *string { return s.EstimatedPriceRange }},
		{name: "short description",
			set: func(s *suggestion.Suggestion, v *string) { s.ShortDescription = v },
			get: func(s *suggestion.Suggestion) *string { return s.ShortDescription }},
		{name: "note",
			set: func(s *suggestion.Suggestion, v *string) { s.Note = v },
			get: func(s *suggestion.Suggestion) *string { return s.Note }},
	}

	tests := []struct {
		name  string
		input *string
		want  *string
	}{
		{name: "absent stays absent", input: nil, want: nil},
		{name: "surrounding spaces are removed", input: ptr("  Jl. Sabang 12  "), want: ptr("Jl. Sabang 12")},
		{name: "tabs and newlines are removed", input: ptr("\tJl. Sabang 12\n"), want: ptr("Jl. Sabang 12")},
		{name: "inner line breaks are kept", input: ptr("Murah.\nPorsi besar."), want: ptr("Murah.\nPorsi besar.")},
		{name: "empty is stored as absent", input: ptr(""), want: nil},
		{name: "blank is stored as absent", input: ptr(" \t\n "), want: nil},
	}

	for _, f := range fields {
		for _, tc := range tests {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				s := submitted()
				f.set(s, tc.input)

				stored, err := createSuggestion(t, s)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				got := f.get(stored)
				switch {
				case tc.want == nil && got != nil:
					t.Errorf("stored %s = %q, want none", f.name, *got)
				case tc.want != nil && got == nil:
					t.Errorf("stored no %s, want %q", f.name, *tc.want)
				case tc.want != nil && *got != *tc.want:
					t.Errorf("stored %s = %q, want %q", f.name, *got, *tc.want)
				}
			})
		}
	}
}

func TestService_Create_OptionalTextLimits(t *testing.T) {
	fields := []struct {
		name    string
		limit   int
		tooLong error
		set     func(*suggestion.Suggestion, *string)
	}{
		{name: "address", limit: place.MaxAddressLength, tooLong: suggestion.ErrAddressTooLong,
			set: func(s *suggestion.Suggestion, v *string) { s.Address = v }},
		{name: "operating hours", limit: place.MaxOperatingHoursLength, tooLong: suggestion.ErrOperatingHoursTooLong,
			set: func(s *suggestion.Suggestion, v *string) { s.OperatingHours = v }},
		{name: "estimated price range", limit: place.MaxEstimatedPriceRangeLength, tooLong: suggestion.ErrEstimatedPriceRangeTooLong,
			set: func(s *suggestion.Suggestion, v *string) { s.EstimatedPriceRange = v }},
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
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if tc.wantErr != nil && stored != nil {
					t.Fatal("refused suggestion reached storage")
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
		{name: "1001 characters", note: thousand + "a", wantErr: suggestion.ErrNoteTooLong},
		{name: "1001 multibyte characters", note: strings.Repeat("é", 1001), wantErr: suggestion.ErrNoteTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := submitted()
			s.Note = ptr(tc.note)

			stored, err := createSuggestion(t, s)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && stored != nil {
				t.Fatal("too long note reached storage")
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
				if !errors.Is(err, suggestion.ErrInvalidAttributes) {
					t.Fatalf("err = %v, want ErrInvalidAttributes", err)
				}
				if stored != nil {
					t.Fatalf("rejected attributes reached storage as %s", stored.Attributes)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSON(t, stored.Attributes, tc.wantStored)
		})
	}
}

func TestService_Create_StartsPending(t *testing.T) {
	// A user cannot submit a suggestion that is already reviewed.
	s := submitted()
	s.Status = suggestion.StatusApproved
	s.RejectionReason = ptr("Tutup")
	s.PlaceID = ptr(int64(42))
	s.ReviewedBy = ptr(int64(99))
	s.ReviewedAt = ptr(time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC))

	stored, err := createSuggestion(t, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Status != suggestion.StatusPending {
		t.Errorf("stored status = %q, want %q", stored.Status, suggestion.StatusPending)
	}
	if stored.RejectionReason != nil || stored.PlaceID != nil || stored.ReviewedBy != nil || stored.ReviewedAt != nil {
		t.Errorf("review fields reached storage: reason %v, place %v, reviewer %v, reviewed at %v",
			stored.RejectionReason, stored.PlaceID, stored.ReviewedBy, stored.ReviewedAt)
	}
}

func TestService_Create_KeepsAuthorAndCatalog(t *testing.T) {
	stored, err := createSuggestion(t, submitted())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.UserID != 7 || stored.SubDistrictID != 1 || stored.CategoryID != 2 {
		t.Errorf("stored user %d, sub-district %d, category %d; want user 7, sub-district 1, category 2",
			stored.UserID, stored.SubDistrictID, stored.CategoryID)
	}
}

func TestService_Create_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name     string
		storeErr error // returned when the suggestion is stored
		wantErr  error // nil means the call succeeds
	}{
		{name: "storage reports too many pending suggestions", storeErr: suggestion.ErrTooManyPending, wantErr: suggestion.ErrTooManyPending},
		{name: "storage reports an unknown sub-district", storeErr: suggestion.ErrSubDistrictNotFound, wantErr: suggestion.ErrSubDistrictNotFound},
		{name: "storage reports an unknown category", storeErr: suggestion.ErrCategoryNotFound, wantErr: suggestion.ErrCategoryNotFound},
		{name: "storage failure is returned", storeErr: boom, wantErr: boom},
		{name: "stored suggestion is returned", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&suggestion.Suggestion{ID: 3}, tc.storeErr)

			got, err := suggestion.NewService(repo, noPlaces(t)).Create(context.Background(), submitted())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (got == nil || got.ID != 3) {
				t.Errorf("returned %+v, want the stored suggestion with ID 3", got)
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
		{name: "unknown suggestion", repoErr: suggestion.ErrSuggestionNotFound},
		{name: "storage failure is returned", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), tc.repoErr)

			got, err := suggestion.NewService(repo, noPlaces(t)).GetSuggestion(context.Background(), 3)
			if !errors.Is(err, tc.repoErr) {
				t.Fatalf("err = %v, want %v", err, tc.repoErr)
			}
			if tc.repoErr == nil && (got == nil || got.ID != 3) {
				t.Errorf("got %+v, want the stored suggestion 3", got)
			}
		})
	}
}

func TestService_GetByUser(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name        string
		suggestions []suggestion.Suggestion // returned by storage
		repoErr     error                   // returned by storage
		wantIDs     []int64
	}{
		{name: "suggestions keep the stored order", suggestions: []suggestion.Suggestion{{ID: 5}, {ID: 2}, {ID: 9}}, repoErr: nil, wantIDs: []int64{5, 2, 9}},
		{name: "no suggestions", suggestions: []suggestion.Suggestion{}, repoErr: nil, wantIDs: []int64{}},
		{name: "storage failure", suggestions: nil, repoErr: boom, wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetByUser(gomock.Any(), int64(7)).Return(tc.suggestions, tc.repoErr)

			got, err := suggestion.NewService(repo, noPlaces(t)).GetByUser(context.Background(), 7)
			if !errors.Is(err, tc.repoErr) {
				t.Fatalf("err = %v, want %v", err, tc.repoErr)
			}
			if tc.repoErr != nil {
				return
			}

			ids := make([]int64, 0, len(got))
			for _, s := range got {
				ids = append(ids, s.ID)
			}
			if !slices.Equal(ids, tc.wantIDs) {
				t.Errorf("suggestion IDs = %v, want %v", ids, tc.wantIDs)
			}
		})
	}
}

func TestService_GetMany_Status(t *testing.T) {
	tests := []struct {
		name    string
		status  suggestion.Status
		wantErr error
	}{
		{name: "no status lists every suggestion", status: "", wantErr: nil},
		{name: "pending", status: suggestion.StatusPending, wantErr: nil},
		{name: "approved", status: suggestion.StatusApproved, wantErr: nil},
		{name: "rejected", status: suggestion.StatusRejected, wantErr: nil},
		{name: "unknown status", status: "deleted", wantErr: suggestion.ErrInvalidStatus},
		{name: "status is case sensitive", status: "PENDING", wantErr: suggestion.ErrInvalidStatus},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filter := suggestion.SuggestionFilter{Status: tc.status}

			repo := mock.NewMockRepository(gomock.NewController(t))
			if tc.wantErr == nil {
				repo.EXPECT().GetMany(gomock.Any(), filter).Return([]suggestion.Suggestion{}, nil)
			}

			_, err := suggestion.NewService(repo, noPlaces(t)).GetMany(context.Background(), filter)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestService_GetMany_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return(nil, boom)

	_, err := suggestion.NewService(repo, noPlaces(t)).GetMany(context.Background(),
		suggestion.SuggestionFilter{Status: suggestion.StatusPending})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

// -----------------------------------------------------------------------------
// Approving suggestions
// -----------------------------------------------------------------------------

const adminID int64 = 99

// pending is suggestion 3, stored and waiting for review.
func pending() *suggestion.Suggestion {
	s := submitted()
	s.ID = 3
	s.Status = suggestion.StatusPending
	s.Attributes = json.RawMessage(`{}`)
	s.CreatedAt = time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC)
	return s
}

// approved is suggestion 3 after admin 99 approved it as place 42.
func approved() *suggestion.Suggestion {
	s := pending()
	s.Status = suggestion.StatusApproved
	s.PlaceID = ptr(int64(42))
	s.ReviewedBy = ptr(adminID)
	s.ReviewedAt = ptr(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

// rejected is suggestion 3 after admin 99 rejected it.
func rejected() *suggestion.Suggestion {
	s := pending()
	s.Status = suggestion.StatusRejected
	s.RejectionReason = ptr("Sudah tutup")
	s.ReviewedBy = ptr(adminID)
	s.ReviewedAt = ptr(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC))
	return s
}

// finalPlace is the place the admin approves suggestion 3 as, after
// correcting it.
func finalPlace() *place.Place {
	return &place.Place{
		SubDistrictID: 1,
		CategoryID:    2,
		Name:          "Warung Bu Ani",
		Address:       ptr("Jl. Sabang 12"),
		Location:      postgis.MustPoint(106.8, -6.2),
		Attributes:    json.RawMessage(`{"halal":true}`),
	}
}

func TestService_Approve(t *testing.T) {
	repo, places := newMocks(t)
	want := approved()

	gomock.InOrder(
		repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), nil),
		places.EXPECT().Create(gomock.Any(), finalPlace()).Return(&place.Place{ID: 42}, nil),
		repo.EXPECT().MarkApproved(gomock.Any(), int64(3), int64(42), adminID).Return(want, nil),
	)

	got, err := suggestion.NewService(repo, places).Approve(context.Background(), 3, adminID, finalPlace())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want the approved suggestion %+v", got, want)
	}
}

func TestService_Approve_PlaceWithoutLocation(t *testing.T) {
	// Location is required for suggestions, so the approved place must have
	// one too. Nothing may be read, created, or marked.
	repo, places := newMocks(t)
	p := finalPlace()
	p.Location = nil

	_, err := suggestion.NewService(repo, places).Approve(context.Background(), 3, adminID, p)
	if !errors.Is(err, suggestion.ErrInvalidLocation) {
		t.Fatalf("err = %v, want ErrInvalidLocation", err)
	}
}

func TestService_Approve_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name   string
		getErr error
	}{
		{name: "unknown suggestion", getErr: suggestion.ErrSuggestionNotFound},
		{name: "lookup failure is returned", getErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No place may be created for a suggestion that cannot be read.
			repo, places := newMocks(t)
			repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(nil, tc.getErr)

			_, err := suggestion.NewService(repo, places).Approve(context.Background(), 3, adminID, finalPlace())
			if !errors.Is(err, tc.getErr) {
				t.Fatalf("err = %v, want %v", err, tc.getErr)
			}
		})
	}
}

func TestService_Approve_AlreadyReviewed(t *testing.T) {
	tests := []struct {
		name     string
		existing *suggestion.Suggestion
	}{
		{name: "already approved", existing: approved()},
		{name: "already rejected", existing: rejected()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No place may be created for a reviewed suggestion.
			repo, places := newMocks(t)
			repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(tc.existing, nil)

			_, err := suggestion.NewService(repo, places).Approve(context.Background(), 3, adminID, finalPlace())
			if !errors.Is(err, suggestion.ErrAlreadyReviewed) {
				t.Fatalf("err = %v, want ErrAlreadyReviewed", err)
			}
		})
	}
}

func TestService_Approve_PlaceRejected(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		createErr error
	}{
		{name: "invalid name", createErr: place.ErrInvalidName},
		{name: "invalid attributes", createErr: place.ErrInvalidAttributes},
		{name: "unknown category", createErr: place.ErrCategoryNotFound},
		{name: "unknown sub-district", createErr: place.ErrSubDistrictNotFound},
		{name: "place storage failure", createErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The suggestion stays pending: it may not be marked approved.
			repo, places := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), nil),
				places.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, tc.createErr),
			)

			_, err := suggestion.NewService(repo, places).Approve(context.Background(), 3, adminID, finalPlace())
			if !errors.Is(err, tc.createErr) {
				t.Fatalf("err = %v, want %v", err, tc.createErr)
			}
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
		{name: "another admin reviewed it meanwhile", markErr: suggestion.ErrAlreadyReviewed, deleteErr: nil},
		{name: "suggestion deleted meanwhile", markErr: suggestion.ErrSuggestionNotFound, deleteErr: nil},
		{name: "storage failure is returned", markErr: boom, deleteErr: nil},
		{name: "created place delete failure is ignored", markErr: suggestion.ErrAlreadyReviewed, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The created place is removed, so no place exists without an
			// approved suggestion.
			repo, places := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetSuggestion(gomock.Any(), int64(3)).Return(pending(), nil),
				places.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&place.Place{ID: 42}, nil),
				repo.EXPECT().MarkApproved(gomock.Any(), int64(3), int64(42), adminID).Return(nil, tc.markErr),
				places.EXPECT().Delete(gomock.Any(), int64(42)).Return(tc.deleteErr),
			)

			_, err := suggestion.NewService(repo, places).Approve(context.Background(), 3, adminID, finalPlace())
			if !errors.Is(err, tc.markErr) {
				t.Fatalf("err = %v, want %v", err, tc.markErr)
			}
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
		{name: "surrounding spaces are removed", reason: ptr("  Sudah tutup  "), want: ptr("Sudah tutup")},
		{name: "inner line breaks are kept", reason: ptr("Sudah tutup.\nPindah."), want: ptr("Sudah tutup.\nPindah.")},
		{name: "empty reason is stored as absent", reason: ptr(""), want: nil},
		{name: "blank reason is stored as absent", reason: ptr(" \t\n "), want: nil},
		{name: "exactly 1000 characters", reason: ptr(thousand), want: ptr(thousand)},
		{name: "1000 multibyte characters", reason: ptr(strings.Repeat("é", 1000)), want: ptr(strings.Repeat("é", 1000))},
		{name: "surrounding spaces do not count toward the limit", reason: ptr(" " + thousand + " "), want: ptr(thousand)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))

			var stored *string
			called := false
			repo.EXPECT().Reject(gomock.Any(), int64(3), adminID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _, _ int64, reason *string) (*suggestion.Suggestion, error) {
					called, stored = true, reason
					return rejected(), nil
				})

			_, err := suggestion.NewService(repo, noPlaces(t)).Reject(context.Background(), 3, adminID, tc.reason)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !called {
				t.Fatal("rejection was not stored")
			}
			switch {
			case tc.want == nil && stored != nil:
				t.Errorf("stored reason = %q, want none", *stored)
			case tc.want != nil && stored == nil:
				t.Errorf("stored no reason, want %q", *tc.want)
			case tc.want != nil && *stored != *tc.want:
				t.Errorf("stored reason = %q, want %q", *stored, *tc.want)
			}
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

			_, err := suggestion.NewService(repo, noPlaces(t)).Reject(context.Background(), 3, adminID, ptr(tc.reason))
			if !errors.Is(err, suggestion.ErrReasonTooLong) {
				t.Fatalf("err = %v, want ErrReasonTooLong", err)
			}
		})
	}
}

func TestService_Reject_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		rejectErr error
	}{
		{name: "unknown suggestion", rejectErr: suggestion.ErrSuggestionNotFound},
		{name: "already reviewed", rejectErr: suggestion.ErrAlreadyReviewed},
		{name: "storage failure is returned", rejectErr: boom},
		{name: "rejected suggestion is returned", rejectErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := rejected()

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Reject(gomock.Any(), int64(3), adminID, gomock.Any()).Return(want, tc.rejectErr)

			got, err := suggestion.NewService(repo, noPlaces(t)).Reject(context.Background(), 3, adminID, ptr("Sudah tutup"))
			if !errors.Is(err, tc.rejectErr) {
				t.Fatalf("err = %v, want %v", err, tc.rejectErr)
			}
			if tc.rejectErr == nil && got != want {
				t.Errorf("got %+v, want the rejected suggestion %+v", got, want)
			}
		})
	}
}
