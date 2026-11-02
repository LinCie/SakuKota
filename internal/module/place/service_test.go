package place_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/httpx"
	"rantau-go/internal/module/place"
	"rantau-go/internal/module/place/mock"
	"rantau-go/internal/postgis"
	"rantau-go/internal/storage"
)

// warungCategory has one required field and one optional field of each data
// type that attributes can hold.
func warungCategory() *place.Category {
	return &place.Category{
		ID:   1,
		Name: "Warung",
		Slug: "warung",
		Fields: []place.CategoryField{
			{Key: "contact", Label: "Contact", DataType: place.FieldTypeText, Required: true},
			{Key: "seats", Label: "Seats", DataType: place.FieldTypeNumber},
			{Key: "halal", Label: "Halal", DataType: place.FieldTypeBoolean},
			{Key: "payment", Label: "Payment", DataType: place.FieldTypeSelect, Options: place.FieldOptions{"cash", "card"}},
			{Key: "price_range", Label: "Price range", DataType: place.FieldTypeRange},
			{Key: "opened_on", Label: "Opened on", DataType: place.FieldTypeDate},
		},
	}
}

// openCategory has no required fields.
func openCategory() *place.Category {
	return &place.Category{
		ID:   2,
		Name: "Kopi",
		Slug: "kopi",
		Fields: []place.CategoryField{
			{Key: "seats", Label: "Seats", DataType: place.FieldTypeNumber},
			{Key: "halal", Label: "Halal", DataType: place.FieldTypeBoolean},
		},
	}
}

// cafeCategory has no required fields. Its optional text field "website" is
// the text attribute the blank-as-absent rule applies to.
func cafeCategory() *place.Category {
	return &place.Category{
		ID:   3,
		Name: "Cafe",
		Slug: "cafe",
		Fields: []place.CategoryField{
			{Key: "website", Label: "Website", DataType: place.FieldTypeText},
			{Key: "seats", Label: "Seats", DataType: place.FieldTypeNumber},
		},
	}
}

// createPlace runs Service.Create for p in category c. It returns the place the
// repository was asked to store, or nil if storage was never reached.
func createPlace(t *testing.T, c *place.Category, p *place.Place) (*place.Place, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).Return(c, nil).AnyTimes()

	var stored *place.Place
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *place.Place) (*place.Place, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := place.NewService(repo, noImages(t)).Create(context.Background(), p)
	return stored, err
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

// noImages returns image storage that fails the test if it is used.
func noImages(t *testing.T) *mock.MockImageStorage {
	t.Helper()
	return mock.NewMockImageStorage(gomock.NewController(t))
}

// newMocks returns a repository and image storage that fail the test on any
// call they do not expect.
func newMocks(t *testing.T) (*mock.MockRepository, *mock.MockImageStorage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mock.NewMockRepository(ctrl), mock.NewMockImageStorage(ctrl)
}

// -----------------------------------------------------------------------------
// Creating places
// -----------------------------------------------------------------------------

func TestService_Create_Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  bool
	}{
		{name: "surrounding spaces are removed", input: "  Warung Bu Ani  ", wantName: "Warung Bu Ani", wantErr: false},
		{name: "tabs and newlines are removed", input: "\tWarung Bu Ani\n", wantName: "Warung Bu Ani", wantErr: false},
		{name: "inner spacing is kept", input: "Warung  Bu Ani", wantName: "Warung  Bu Ani", wantErr: false},
		{name: "empty", input: "", wantName: "", wantErr: true},
		{name: "spaces only", input: "   ", wantName: "", wantErr: true},
		{name: "tabs and newlines only", input: "\t\n", wantName: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, openCategory(), &place.Place{
				Name: tc.input, SubDistrictID: 1, CategoryID: 2,
			})
			if tc.wantErr {
				require.ErrorIs(t, err, place.ErrInvalidName)
				require.Nil(t, stored, "blank name reached storage")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, stored.Name)
		})
	}
}

func TestService_Create_NameTooLong(t *testing.T) {
	_, err := createPlace(t, openCategory(), &place.Place{
		Name: strings.Repeat("a", 151), SubDistrictID: 1, CategoryID: 2,
	})
	require.ErrorIs(t, err, place.ErrNameTooLong)
}

func TestService_Create_Attributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string // stored JSON when the attributes are accepted
		wantErr    bool
	}{
		{name: "required field only", attributes: `{"contact":"0812-3456"}`, wantStored: `{"contact":"0812-3456"}`, wantErr: false},
		{
			name:       "every type with a valid value",
			attributes: `{"contact":"0812","seats":12,"halal":true,"payment":"card","price_range":{"min":10000,"max":50000},"opened_on":"2024-01-31"}`,
			wantStored: `{"contact":"0812","seats":12,"halal":true,"payment":"card","price_range":{"min":10000,"max":50000},"opened_on":"2024-01-31"}`,
			wantErr:    false,
		},
		{name: "range with equal bounds", attributes: `{"contact":"x","price_range":{"min":5,"max":5}}`, wantStored: `{"contact":"x","price_range":{"min":5,"max":5}}`, wantErr: false},
		{name: "integer keeps every digit", attributes: `{"contact":"x","seats":12345678901234567890}`, wantStored: `{"contact":"x","seats":12345678901234567890}`, wantErr: false},
		{name: "decimal keeps its trailing zero", attributes: `{"contact":"x","seats":1.50}`, wantStored: `{"contact":"x","seats":1.50}`, wantErr: false},
		{
			name:       "text is kept verbatim",
			attributes: `{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`,
			wantStored: `{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`,
			wantErr:    false,
		},
		{name: "optional null is dropped", attributes: `{"contact":"x","seats":null,"halal":null}`, wantStored: `{"contact":"x"}`, wantErr: false},

		{name: "required field missing", attributes: `{"seats":1}`, wantStored: "", wantErr: true},
		{name: "required text is only spaces", attributes: `{"contact":"   "}`, wantStored: "", wantErr: true},
		{name: "required field is null", attributes: `{"contact":null}`, wantStored: "", wantErr: true},
		{name: "null attributes miss the required field", attributes: `null`, wantStored: "", wantErr: true},
		{name: "empty object misses the required field", attributes: `{}`, wantStored: "", wantErr: true},
		{name: "array instead of object", attributes: `[1,2]`, wantStored: "", wantErr: true},
		{name: "string instead of object", attributes: `"text"`, wantStored: "", wantErr: true},
		{name: "malformed JSON", attributes: `{"contact":`, wantStored: "", wantErr: true},
		{name: "unknown key", attributes: `{"contact":"x","parking":true}`, wantStored: "", wantErr: true},
		{name: "text given a number", attributes: `{"contact":5}`, wantStored: "", wantErr: true},
		{name: "number given a string", attributes: `{"contact":"x","seats":"12"}`, wantStored: "", wantErr: true},
		{name: "number given a boolean", attributes: `{"contact":"x","seats":true}`, wantStored: "", wantErr: true},
		{name: "boolean given a string", attributes: `{"contact":"x","halal":"true"}`, wantStored: "", wantErr: true},
		{name: "boolean given a number", attributes: `{"contact":"x","halal":1}`, wantStored: "", wantErr: true},
		{name: "select value outside its options", attributes: `{"contact":"x","payment":"bitcoin"}`, wantStored: "", wantErr: true},
		{name: "select given a number", attributes: `{"contact":"x","payment":1}`, wantStored: "", wantErr: true},
		{name: "range with min above max", attributes: `{"contact":"x","price_range":{"min":9,"max":1}}`, wantStored: "", wantErr: true},
		{name: "range missing max", attributes: `{"contact":"x","price_range":{"min":1}}`, wantStored: "", wantErr: true},
		{name: "range with an extra key", attributes: `{"contact":"x","price_range":{"min":1,"max":2,"step":1}}`, wantStored: "", wantErr: true},
		{name: "range with string bounds", attributes: `{"contact":"x","price_range":{"min":"1","max":"2"}}`, wantStored: "", wantErr: true},
		{name: "range given as an array", attributes: `{"contact":"x","price_range":[1,2]}`, wantStored: "", wantErr: true},
		{name: "date with slashes", attributes: `{"contact":"x","opened_on":"2024/01/31"}`, wantStored: "", wantErr: true},
		{name: "date that does not exist", attributes: `{"contact":"x","opened_on":"2024-02-30"}`, wantStored: "", wantErr: true},
		{name: "date given as a number", attributes: `{"contact":"x","opened_on":20240131}`, wantStored: "", wantErr: true},
		{name: "date with a time", attributes: `{"contact":"x","opened_on":"2024-01-31T00:00:00Z"}`, wantStored: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, warungCategory(), &place.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			if tc.wantErr {
				require.ErrorIs(t, err, place.ErrInvalidAttributes)
				require.Nil(t, stored, "rejected attributes reached storage")
				return
			}
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantStored)
		})
	}
}

func TestService_Create_AttributesWithoutRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string
	}{
		{name: "absent attributes are stored as an empty object", attributes: "", wantStored: `{}`},
		{name: "null attributes are stored as an empty object", attributes: `null`, wantStored: `{}`},
		{name: "empty object is kept", attributes: `{}`, wantStored: `{}`},
		{name: "set optional field is kept", attributes: `{"seats":5}`, wantStored: `{"seats":5}`},
		{name: "null optional field is dropped", attributes: `{"halal":null}`, wantStored: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, openCategory(), &place.Place{
				Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2,
				Attributes: json.RawMessage(tc.attributes),
			})
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantStored)
		})
	}
}

func TestService_Create_BlankOptionalText(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string
	}{
		{name: "empty string is dropped", attributes: `{"website":""}`, wantStored: `{}`},
		{name: "spaces only is dropped", attributes: `{"website":"   "}`, wantStored: `{}`},
		{name: "tabs and newlines only is dropped", attributes: `{"website":"\t\n"}`, wantStored: `{}`},
		{name: "blank key is removed and other keys are kept", attributes: `{"website":"","seats":5}`, wantStored: `{"seats":5}`},
		{name: "non-blank text is kept verbatim", attributes: `{"website":"  Halal  "}`, wantStored: `{"website":"  Halal  "}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).Return(cafeCategory(), nil).AnyTimes()

			var stored *place.Place
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, got *place.Place) (*place.Place, error) {
					stored = got
					return got, nil
				})

			_, err := place.NewService(repo, noImages(t)).Create(context.Background(), &place.Place{
				Name: "Kafe Senja", SubDistrictID: 1, CategoryID: 3,
				Attributes: json.RawMessage(tc.attributes),
			})
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantStored)
		})
	}
}

func TestService_Create_BlankAttributeRejected(t *testing.T) {
	// Required text may not be blank, and blank is not absent for number,
	// boolean, select, range, or date fields.
	tests := []struct {
		name       string
		attributes string
	}{
		{name: "required text is empty", attributes: `{"contact":""}`},
		{name: "required text is only whitespace", attributes: `{"contact":" \t\n "}`},
		{name: "number is empty", attributes: `{"contact":"x","seats":""}`},
		{name: "boolean is empty", attributes: `{"contact":"x","halal":""}`},
		{name: "select is empty", attributes: `{"contact":"x","payment":""}`},
		{name: "range is empty", attributes: `{"contact":"x","price_range":""}`},
		{name: "date is empty", attributes: `{"contact":"x","opened_on":""}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No Create expectation: nothing may be stored.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).Return(warungCategory(), nil).AnyTimes()

			_, err := place.NewService(repo, noImages(t)).Create(context.Background(), &place.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			require.ErrorIs(t, err, place.ErrInvalidAttributes)
		})
	}
}

func TestService_Create_Location(t *testing.T) {
	tests := []struct {
		name     string
		lon, lat float64
		omit     bool // no location in the request
		wantErr  bool
	}{
		{name: "Jakarta", lon: 106.8, lat: -6.2, omit: false, wantErr: false},
		{name: "Sabang, the western tip", lon: 95.3, lat: 5.9, omit: false, wantErr: false},
		{name: "Merauke, the eastern tip", lon: 140.4, lat: -8.5, omit: false, wantErr: false},
		{name: "west edge of the box", lon: 95, lat: -6.2, omit: false, wantErr: false},
		{name: "east edge of the box", lon: 141, lat: -6.2, omit: false, wantErr: false},
		{name: "north edge of the box", lon: 106.8, lat: 6, omit: false, wantErr: false},
		{name: "south edge of the box", lon: 106.8, lat: -11, omit: false, wantErr: false},
		{name: "location omitted", lon: 0, lat: 0, omit: true, wantErr: false},
		{name: "Tokyo", lon: 139.7, lat: 35.7, omit: false, wantErr: true},
		{name: "London", lon: -0.1, lat: 51.5, omit: false, wantErr: true},
		{name: "null island", lon: 0, lat: 0, omit: false, wantErr: true},
		{name: "just west of the box", lon: 94.99, lat: -6.2, omit: false, wantErr: true},
		{name: "just east of the box", lon: 141.01, lat: -6.2, omit: false, wantErr: true},
		{name: "just north of the box", lon: 106.8, lat: 6.01, omit: false, wantErr: true},
		{name: "just south of the box", lon: 106.8, lat: -11.01, omit: false, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &place.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
			if !tc.omit {
				point, err := postgis.NewPoint(tc.lon, tc.lat)
				require.NoError(t, err, "invalid test point")
				p.Location = point
			}

			stored, err := createPlace(t, openCategory(), p)
			if tc.wantErr {
				require.Error(t, err, "location outside Indonesia was accepted")
				require.Nil(t, stored, "location outside Indonesia reached storage")
				return
			}
			require.NoError(t, err)
			if tc.omit {
				assert.Nil(t, stored.Location, "omitted location was stored")
				return
			}
			require.NotNil(t, stored.Location, "stored location is missing")
			assert.Equal(t, tc.lon, stored.Location.X())
			assert.Equal(t, tc.lat, stored.Location.Y())
		})
	}
}

func TestService_Create_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		lookupErr error // returned when the category is looked up
		storeErr  error // returned when the place is stored
		wantErr   error // nil means the call succeeds
	}{
		{name: "unknown category stops before storage", lookupErr: place.ErrCategoryNotFound, storeErr: nil, wantErr: place.ErrCategoryNotFound},
		{name: "storage reports an unknown sub-district", lookupErr: nil, storeErr: place.ErrSubDistrictNotFound, wantErr: place.ErrSubDistrictNotFound},
		{name: "storage reports an unknown category", lookupErr: nil, storeErr: place.ErrCategoryNotFound, wantErr: place.ErrCategoryNotFound},
		{name: "storage failure is returned", lookupErr: nil, storeErr: boom, wantErr: boom},
		{name: "stored place is returned", lookupErr: nil, storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			if tc.lookupErr != nil {
				repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).Return(nil, tc.lookupErr)
			} else {
				repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).Return(openCategory(), nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&place.Place{ID: 7}, tc.storeErr)
			}

			got, err := place.NewService(repo, noImages(t)).Create(context.Background(), &place.Place{
				Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2,
			})
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored place was not returned")
				assert.Equal(t, int64(7), got.ID, "want the stored place's ID")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Updating and deleting places
// -----------------------------------------------------------------------------

// present is a PATCH field the client sent with the value v.
func present[T any](v T) httpx.Optional[T] {
	return httpx.Optional[T]{Set: true, Value: &v}
}

// null is a PATCH field the client sent as null.
func null[T any]() httpx.Optional[T] {
	return httpx.Optional[T]{Set: true}
}

func ptr[T any](v T) *T {
	return &v
}

// savedKopi is the stored place that updates start from: a Kopi in Menteng
// with every optional field set.
func savedKopi() *place.Place {
	return &place.Place{
		ID:                  7,
		SubDistrictID:       1,
		CategoryID:          2,
		Name:                "Kopi Senja",
		Address:             ptr("Jl. Sudirman 9"),
		Location:            postgis.MustPoint(106.8, -6.2),
		OperatingHours:      ptr("08:00-17:00"),
		EstimatedPriceRange: ptr("Rp20.000-50.000"),
		ShortDescription:    ptr("Kopi tubruk"),
		Attributes:          json.RawMessage(`{"halal":true,"seats":12}`),
		Category:            place.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}
}

// lookupCategory answers category lookups for the update tests. Category 1 is
// Warung, which requires "contact". Category 2 is Kopi. Category 3 is Cafe.
func lookupCategory(_ context.Context, id int64) (*place.Category, error) {
	switch id {
	case 1:
		return warungCategory(), nil
	case 2:
		return openCategory(), nil
	case 3:
		return cafeCategory(), nil
	}
	return nil, place.ErrCategoryNotFound
}

// updatePlace runs Service.Update for place id, where existing is the stored
// place. It returns the place the repository was asked to store, or nil if
// storage was never reached.
func updatePlace(t *testing.T, existing *place.Place, id int64, patch place.PlacePatch) (*place.Place, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetPlace(gomock.Any(), id).Return(existing, nil).AnyTimes()
	repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).DoAndReturn(lookupCategory).AnyTimes()

	var stored *place.Place
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *place.Place) (*place.Place, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := place.NewService(repo, noImages(t)).Update(context.Background(), id, patch)
	return stored, err
}

func TestService_Update_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown place", getErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No Update expectation: nothing may be stored when the place cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := place.NewService(repo, noImages(t)).Update(context.Background(), 7,
				place.PlacePatch{Name: present("Kopi Baru")})
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_Update_Fields(t *testing.T) {
	tests := []struct {
		name   string
		patch  place.PlacePatch
		change func(*place.Place) // turns savedKopi into the wanted stored place
	}{
		{name: "empty patch changes nothing", patch: place.PlacePatch{}, change: func(*place.Place) {}},
		{name: "name is trimmed", patch: place.PlacePatch{Name: present("  Kopi Baru\t")},
			change: func(p *place.Place) { p.Name = "Kopi Baru" }},
		{name: "address is replaced", patch: place.PlacePatch{Address: present("Jl. Kemang 2")},
			change: func(p *place.Place) { p.Address = ptr("Jl. Kemang 2") }},
		{name: "location is replaced", patch: place.PlacePatch{Location: present(*postgis.MustPoint(106.85, -6.18))},
			change: func(p *place.Place) { p.Location = postgis.MustPoint(106.85, -6.18) }},
		{name: "sub-district is replaced", patch: place.PlacePatch{SubDistrictID: present(int64(2))},
			change: func(p *place.Place) { p.SubDistrictID = 2 }},
		{name: "null clears every nullable field", patch: place.PlacePatch{
			Address:             null[string](),
			Location:            null[postgis.Point](),
			OperatingHours:      null[string](),
			EstimatedPriceRange: null[string](),
			ShortDescription:    null[string](),
		}, change: func(p *place.Place) {
			p.Address, p.Location, p.OperatingHours = nil, nil, nil
			p.EstimatedPriceRange, p.ShortDescription = nil, nil
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := savedKopi()
			tc.change(want)

			stored, err := updatePlace(t, savedKopi(), 7, tc.patch)
			require.NoError(t, err)
			assert.Equal(t, want, stored)
		})
	}
}

func TestService_Update_Name(t *testing.T) {
	long := strings.Repeat("a", 150)

	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  error
	}{
		{name: "surrounding spaces are removed", input: "  Kopi Baru  ", wantName: "Kopi Baru", wantErr: nil},
		{name: "empty", input: "", wantName: "", wantErr: place.ErrInvalidName},
		{name: "spaces only", input: "   ", wantName: "", wantErr: place.ErrInvalidName},
		{name: "150 characters is allowed", input: long, wantName: long, wantErr: nil},
		{name: "151 characters is too long", input: long + "a", wantName: "", wantErr: place.ErrNameTooLong},
		{name: "150 multibyte characters is allowed", input: strings.Repeat("é", 150), wantName: strings.Repeat("é", 150), wantErr: nil},
		{name: "151 multibyte characters is too long", input: strings.Repeat("é", 151), wantName: "", wantErr: place.ErrNameTooLong},
		{name: "length is counted after trimming", input: "  " + long + "  ", wantName: long, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updatePlace(t, savedKopi(), 7, place.PlacePatch{Name: present(tc.input)})
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

func TestService_Update_Location(t *testing.T) {
	tests := []struct {
		name    string
		lon     float64
		lat     float64
		wantErr error
	}{
		{name: "inside Indonesia", lon: 106.8, lat: -6.2, wantErr: nil},
		{name: "on the western edge", lon: 95, lat: -2, wantErr: nil},
		{name: "west of the Indonesia box", lon: 94.9, lat: -2, wantErr: place.ErrInvalidLocation},
		{name: "north of the Indonesia box", lon: 110, lat: 6.5, wantErr: place.ErrInvalidLocation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updatePlace(t, savedKopi(), 7,
				place.PlacePatch{Location: present(*postgis.MustPoint(tc.lon, tc.lat))})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "location outside Indonesia reached storage")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored.Location, "stored location is missing")
			assert.Equal(t, tc.lon, stored.Location.X())
			assert.Equal(t, tc.lat, stored.Location.Y())
		})
	}
}

func TestService_Update_Attributes(t *testing.T) {
	// The place is a Kopi, which has no required fields.
	tests := []struct {
		name      string
		attrs     httpx.Optional[json.RawMessage]
		wantAttrs string
		wantErr   error
	}{
		{name: "new attributes replace the old ones", attrs: present(json.RawMessage(`{"halal":false}`)), wantAttrs: `{"halal":false}`, wantErr: nil},
		{name: "null attributes become an empty object", attrs: null[json.RawMessage](), wantAttrs: `{}`, wantErr: nil},
		{name: "unknown key is rejected", attrs: present(json.RawMessage(`{"parking":true}`)), wantAttrs: "", wantErr: place.ErrInvalidAttributes},
		{name: "wrong type is rejected", attrs: present(json.RawMessage(`{"seats":"many"}`)), wantAttrs: "", wantErr: place.ErrInvalidAttributes},
		{name: "array is rejected", attrs: present(json.RawMessage(`[1]`)), wantAttrs: "", wantErr: place.ErrInvalidAttributes},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updatePlace(t, savedKopi(), 7, place.PlacePatch{Attributes: tc.attrs})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "invalid attributes reached storage")
				return
			}
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantAttrs)
		})
	}
}

func TestService_Update_BlankOptionalText(t *testing.T) {
	tests := []struct {
		name      string
		stored    string // attributes of the place before the update
		patch     string // attributes sent in the update
		wantAttrs string
	}{
		{name: "empty string is dropped", stored: `{"seats":12}`, patch: `{"website":"","seats":12}`, wantAttrs: `{"seats":12}`},
		{name: "spaces only is dropped", stored: `{"seats":12}`, patch: `{"website":"   ","seats":12}`, wantAttrs: `{"seats":12}`},
		{name: "tabs and newlines only is dropped", stored: `{"seats":12}`, patch: `{"website":"\t\n"}`, wantAttrs: `{}`},
		{name: "stored text is removed when the patch blanks it", stored: `{"website":"https://a.example","seats":12}`, patch: `{"website":"","seats":12}`, wantAttrs: `{"seats":12}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing := savedKopi()
			existing.CategoryID = 3
			existing.Category = place.Category{ID: 3, Name: "Cafe", Slug: "cafe"}
			existing.Attributes = json.RawMessage(tc.stored)

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(existing, nil).AnyTimes()
			repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).DoAndReturn(lookupCategory).AnyTimes()

			var stored *place.Place
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, got *place.Place) (*place.Place, error) {
					stored = got
					return got, nil
				})

			_, err := place.NewService(repo, noImages(t)).Update(context.Background(), 7,
				place.PlacePatch{Attributes: present(json.RawMessage(tc.patch))})
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantAttrs)
		})
	}
}

func TestService_Update_BlankRequiredText(t *testing.T) {
	for _, patch := range []string{`{"contact":""}`, `{"contact":"   "}`, `{"contact":"\t\n"}`} {
		t.Run(patch, func(t *testing.T) {
			// The place is a Warung, which requires "contact". No Update
			// expectation: nothing may be stored.
			existing := savedKopi()
			existing.CategoryID = 1
			existing.Category = place.Category{ID: 1, Name: "Warung", Slug: "warung"}
			existing.Attributes = json.RawMessage(`{"contact":"0812"}`)

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(existing, nil).AnyTimes()
			repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).DoAndReturn(lookupCategory).AnyTimes()

			_, err := place.NewService(repo, noImages(t)).Update(context.Background(), 7,
				place.PlacePatch{Attributes: present(json.RawMessage(patch))})
			require.ErrorIs(t, err, place.ErrInvalidAttributes)
		})
	}
}

func TestService_Update_Category(t *testing.T) {
	tests := []struct {
		name           string
		attrs          string // the place's attributes before the update
		patch          place.PlacePatch
		wantCategoryID int64
		wantAttrs      string
		wantErr        error
	}{
		{
			name:           "attributes that fit the new category are kept",
			attrs:          `{"contact":"0812","seats":12}`,
			patch:          place.PlacePatch{CategoryID: present(int64(1))},
			wantCategoryID: 1,
			wantAttrs:      `{"contact":"0812","seats":12}`,
			wantErr:        nil,
		},
		{
			name:           "attributes missing a required field of the new category are rejected",
			attrs:          `{"halal":true,"seats":12}`,
			patch:          place.PlacePatch{CategoryID: present(int64(1))},
			wantCategoryID: 0,
			wantAttrs:      "",
			wantErr:        place.ErrInvalidAttributes,
		},
		{
			name:           "attributes unknown to the new category are rejected",
			attrs:          `{"contact":"0812"}`,
			patch:          place.PlacePatch{CategoryID: present(int64(2))},
			wantCategoryID: 0,
			wantAttrs:      "",
			wantErr:        place.ErrInvalidAttributes,
		},
		{
			name:           "new attributes are checked against the new category",
			attrs:          `{"contact":"0812"}`,
			patch:          place.PlacePatch{CategoryID: present(int64(1)), Attributes: present(json.RawMessage(`{"seats":12}`))},
			wantCategoryID: 0,
			wantAttrs:      "",
			wantErr:        place.ErrInvalidAttributes,
		},
		{
			name:           "new attributes that fit the new category are stored",
			attrs:          `{"halal":true}`,
			patch:          place.PlacePatch{CategoryID: present(int64(1)), Attributes: present(json.RawMessage(`{"contact":"0812"}`))},
			wantCategoryID: 1,
			wantAttrs:      `{"contact":"0812"}`,
			wantErr:        nil,
		},
		{
			name:           "unknown category is rejected",
			attrs:          `{"halal":true}`,
			patch:          place.PlacePatch{CategoryID: present(int64(99))},
			wantCategoryID: 0,
			wantAttrs:      "",
			wantErr:        place.ErrCategoryNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing := savedKopi()
			existing.Attributes = json.RawMessage(tc.attrs)

			stored, err := updatePlace(t, existing, 7, tc.patch)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "rejected update reached storage")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantCategoryID, stored.CategoryID)
			assertJSON(t, stored.Attributes, tc.wantAttrs)
		})
	}
}

func TestService_Update_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		updateErr error // returned when the place is stored
		wantErr   error // nil means the call succeeds
	}{
		{name: "storage reports the place is gone", updateErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "storage reports an unknown sub-district", updateErr: place.ErrSubDistrictNotFound, wantErr: place.ErrSubDistrictNotFound},
		{name: "storage failure is returned", updateErr: boom, wantErr: boom},
		{name: "stored place is returned", updateErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil)
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&place.Place{ID: 7}, tc.updateErr)

			got, err := place.NewService(repo, noImages(t)).Update(context.Background(), 7,
				place.PlacePatch{Name: present("Kopi Baru")})
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored place was not returned")
				assert.Equal(t, int64(7), got.ID, "want the stored place's ID")
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "deletes the place", repoErr: nil, wantErr: nil},
		{name: "unknown place", repoErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The place has no image, so no image may be deleted.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := place.NewService(repo, noImages(t)).Delete(context.Background(), 7)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_Delete_Image(t *testing.T) {
	// The repository queues the image for deletion, so the service leaves it
	// to the cleanup worker and must not delete it itself.
	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil)

	err := place.NewService(repo, noImages(t)).Delete(context.Background(), 7)
	require.NoError(t, err, "the place was deleted")
}

// -----------------------------------------------------------------------------
// Optional text fields
// -----------------------------------------------------------------------------

// placeTextField is an optional text field of a place. limit is the length its
// column allows in characters, or 0 when the field has no limit.
type placeTextField struct {
	name     string
	limit    int
	tooLong  error
	set      func(p *place.Place, v *string)
	setPatch func(p *place.PlacePatch, v string)
	get      func(p *place.Place) *string
}

var placeTextFields = []placeTextField{
	{name: "address", limit: place.MaxAddressLength, tooLong: place.ErrAddressTooLong,
		set:      func(p *place.Place, v *string) { p.Address = v },
		setPatch: func(p *place.PlacePatch, v string) { p.Address = present(v) },
		get:      func(p *place.Place) *string { return p.Address }},
	{name: "operating hours", limit: place.MaxOperatingHoursLength, tooLong: place.ErrOperatingHoursTooLong,
		set:      func(p *place.Place, v *string) { p.OperatingHours = v },
		setPatch: func(p *place.PlacePatch, v string) { p.OperatingHours = present(v) },
		get:      func(p *place.Place) *string { return p.OperatingHours }},
	{name: "estimated price range", limit: place.MaxEstimatedPriceRangeLength, tooLong: place.ErrEstimatedPriceRangeTooLong,
		set:      func(p *place.Place, v *string) { p.EstimatedPriceRange = v },
		setPatch: func(p *place.PlacePatch, v string) { p.EstimatedPriceRange = present(v) },
		get:      func(p *place.Place) *string { return p.EstimatedPriceRange }},
	{name: "short description",
		set:      func(p *place.Place, v *string) { p.ShortDescription = v },
		setPatch: func(p *place.PlacePatch, v string) { p.ShortDescription = present(v) },
		get:      func(p *place.Place) *string { return p.ShortDescription }},
}

func TestService_Create_OptionalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{name: "surrounding whitespace is removed", input: "  Jl. Sudirman  ", want: ptr("Jl. Sudirman")},
		{name: "blank is stored as absent", input: " \t\n ", want: nil},
	}

	for _, f := range placeTextFields {
		for _, tc := range tests {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				p := &place.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
				input := tc.input
				f.set(p, &input)

				stored, err := createPlace(t, openCategory(), p)
				require.NoError(t, err)
				assert.Equal(t, ptrString(tc.want), ptrString(f.get(stored)), "stored %s", f.name)
			})
		}
	}
}

func TestService_Create_OptionalTextLimits(t *testing.T) {
	for _, f := range placeTextFields {
		if f.limit == 0 {
			continue
		}

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
				p := &place.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
				input := tc.input
				f.set(p, &input)

				stored, err := createPlace(t, openCategory(), p)
				require.ErrorIs(t, err, tc.wantErr)
				if tc.wantErr != nil {
					require.Nil(t, stored, "refused place reached storage")
				}
			})
		}
	}
}

func TestService_Update_OptionalText(t *testing.T) {
	for _, f := range placeTextFields {
		tests := []struct {
			name    string
			input   string
			want    *string
			wantErr error
		}{
			{name: "surrounding whitespace is removed", input: "  Jl. Sudirman  ", want: ptr("Jl. Sudirman")},
			{name: "blank is stored as absent", input: " \t\n ", want: nil},
		}
		if f.limit > 0 {
			tests = append(tests, struct {
				name    string
				input   string
				want    *string
				wantErr error
			}{name: "one character over the limit is refused", input: strings.Repeat("a", f.limit+1), wantErr: f.tooLong})
		}

		for _, tc := range tests {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				repo := mock.NewMockRepository(gomock.NewController(t))
				repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil)

				var stored *place.Place
				repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, got *place.Place) (*place.Place, error) {
						stored = got
						return got, nil
					}).AnyTimes()

				var patch place.PlacePatch
				f.setPatch(&patch, tc.input)

				_, err := place.NewService(repo, noImages(t)).Update(context.Background(), 7, patch)
				require.ErrorIs(t, err, tc.wantErr)
				if tc.wantErr != nil {
					require.Nil(t, stored, "refused update reached storage")
					return
				}
				assert.Equal(t, ptrString(tc.want), ptrString(f.get(stored)), "stored %s", f.name)
			})
		}
	}
}

// ptrString describes an optional string for failure messages.
func ptrString(s *string) string {
	if s == nil {
		return "<absent>"
	}
	return fmt.Sprintf("%q", *s)
}

// -----------------------------------------------------------------------------
// Place images
// -----------------------------------------------------------------------------

const (
	uploadKey = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	imageKey  = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	oldKey    = "places/7/9f8e7d6c-5b4a-4321-8fed-cba987654321.png"
)

func TestService_ImageUploadURL(t *testing.T) {
	repo, images := newMocks(t)
	upload := storage.Upload{
		Key:             uploadKey,
		PresignedUpload: storage.PresignedUpload{URL: "http://localhost:3900/signed", Method: "PUT"},
	}
	gomock.InOrder(
		repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil),
		images.EXPECT().PresignUpload(gomock.Any(), "places/7", "image/webp").Return(upload, nil),
	)

	got, err := place.NewService(repo, images).ImageUploadURL(context.Background(), 7, "image/webp")
	require.NoError(t, err)
	assert.Equal(t, upload, got)
}

func TestService_ImageUploadURL_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown place", getErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be presigned for a place that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := place.NewService(repo, noImages(t)).ImageUploadURL(context.Background(), 7, "image/webp")
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_ImageUploadURL_PresignFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		presignErr error
	}{
		{name: "invalid image type", presignErr: storage.ErrInvalidImageType},
		{name: "storage failure is returned", presignErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, images := newMocks(t)
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil)
			images.EXPECT().PresignUpload(gomock.Any(), "places/7", "image/gif").Return(storage.Upload{}, tc.presignErr)

			_, err := place.NewService(repo, images).ImageUploadURL(context.Background(), 7, "image/gif")
			require.ErrorIs(t, err, tc.presignErr)
		})
	}
}

// withImage returns savedKopi with the image stored under key, or with no
// image if key is nil.
func withImage(key *string) *place.Place {
	p := savedKopi()
	p.ImageKey = key
	return p
}

func TestService_SetImage(t *testing.T) {
	tests := []struct {
		name   string
		oldKey *string
	}{
		{name: "first image", oldKey: nil},
		{name: "replaces the previous image", oldKey: ptr(oldKey)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues the previous image for deletion, so the
			// service must not delete it itself.
			repo, images := newMocks(t)
			updated := withImage(ptr(imageKey))

			gomock.InOrder(
				repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(withImage(tc.oldKey), nil),
				images.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(imageKey, nil),
				repo.EXPECT().SetImageKey(gomock.Any(), int64(7), imageKey).Return(updated, nil),
			)

			got, err := place.NewService(repo, images).SetImage(context.Background(), 7, uploadKey)
			require.NoError(t, err)
			assert.Same(t, updated, got, "want the stored place")
		})
	}
}

func TestService_SetImage_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown place", getErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be promoted for a place that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := place.NewService(repo, noImages(t)).SetImage(context.Background(), 7, uploadKey)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_SetImage_PromoteFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		promoteErr error
	}{
		{name: "invalid upload key", promoteErr: storage.ErrInvalidImageKey},
		{name: "image not uploaded", promoteErr: storage.ErrImageNotUploaded},
		{name: "invalid image type", promoteErr: storage.ErrInvalidImageType},
		{name: "image too large", promoteErr: storage.ErrImageTooLarge},
		{name: "storage failure is returned", promoteErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be stored, and the old image is kept.
			repo, images := newMocks(t)
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(withImage(ptr(oldKey)), nil)
			images.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return("", tc.promoteErr)

			_, err := place.NewService(repo, images).SetImage(context.Background(), 7, uploadKey)
			require.ErrorIs(t, err, tc.promoteErr)
		})
	}
}

func TestService_SetImage_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		setErr    error
		deleteErr error
	}{
		{name: "place deleted meanwhile", setErr: place.ErrPlaceNotFound},
		{name: "storage failure is returned", setErr: boom},
		{name: "promoted image delete failure is ignored", setErr: boom, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The promoted image is removed, and the old image is kept.
			repo, images := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(withImage(ptr(oldKey)), nil),
				images.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(imageKey, nil),
				repo.EXPECT().SetImageKey(gomock.Any(), int64(7), imageKey).Return(nil, tc.setErr),
				images.EXPECT().Delete(gomock.Any(), imageKey).Return(tc.deleteErr),
			)

			_, err := place.NewService(repo, images).SetImage(context.Background(), 7, uploadKey)
			require.ErrorIs(t, err, tc.setErr)
		})
	}
}

// -----------------------------------------------------------------------------
// Reading a place or a category
// -----------------------------------------------------------------------------

func TestService_GetPlace(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "returns the stored place", repoErr: nil, wantErr: nil},
		{name: "unknown place", repoErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), tc.repoErr)

			got, err := place.NewService(repo, noImages(t)).GetPlace(context.Background(), 7)
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored place was not returned")
				assert.Equal(t, int64(7), got.ID)
			}
		})
	}
}

func TestService_GetCategory(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "returns the stored category with its fields", repoErr: nil, wantErr: nil},
		{name: "unknown category", repoErr: place.ErrCategoryNotFound, wantErr: place.ErrCategoryNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetCategory(gomock.Any(), int64(1)).Return(warungCategory(), tc.repoErr)

			got, err := place.NewService(repo, noImages(t)).GetCategory(context.Background(), 1)
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored category was not returned")
				assert.Equal(t, int64(1), got.ID)
				assert.Len(t, got.Fields, 6, "Warung has 6 fields")
			}
		})
	}
}
