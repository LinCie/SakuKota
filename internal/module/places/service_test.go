package places_test

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

	"rantau-go/internal/module/places"
	"rantau-go/internal/module/places/mock"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// warungCategory has one required field and one optional field of each data
// type that attributes can hold.
func warungCategory() *places.Category {
	return &places.Category{
		ID:   1,
		Name: "Warung",
		Slug: "warung",
		Fields: []places.CategoryField{
			{Key: "contact", Label: "Contact", DataType: places.FieldTypeText, Required: true},
			{Key: "seats", Label: "Seats", DataType: places.FieldTypeNumber},
			{Key: "halal", Label: "Halal", DataType: places.FieldTypeBoolean},
			{Key: "payment", Label: "Payment", DataType: places.FieldTypeSelect, Options: places.FieldOptions{"cash", "card"}},
			{Key: "price_range", Label: "Price range", DataType: places.FieldTypeRange},
			{Key: "opened_on", Label: "Opened on", DataType: places.FieldTypeDate},
		},
	}
}

// openCategory has no required fields.
func openCategory() *places.Category {
	return &places.Category{
		ID:   2,
		Name: "Kopi",
		Slug: "kopi",
		Fields: []places.CategoryField{
			{Key: "seats", Label: "Seats", DataType: places.FieldTypeNumber},
			{Key: "halal", Label: "Halal", DataType: places.FieldTypeBoolean},
		},
	}
}

// cafeCategory has no required fields. Its optional text field "website" is
// the text attribute the blank-as-absent rule applies to.
func cafeCategory() *places.Category {
	return &places.Category{
		ID:   3,
		Name: "Cafe",
		Slug: "cafe",
		Fields: []places.CategoryField{
			{Key: "website", Label: "Website", DataType: places.FieldTypeText},
			{Key: "seats", Label: "Seats", DataType: places.FieldTypeNumber},
		},
	}
}

// createPlace runs Service.Create for p in category c. It returns the place the
// repository was asked to store, or nil if storage was never reached.
func createPlace(t *testing.T, c *places.Category, p *places.Place) (*places.Place, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).Return(c, nil).AnyTimes()

	var stored *places.Place
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *places.Place) (*places.Place, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := places.NewService(repo, noMedia(t)).Create(context.Background(), p)
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

// noMedia returns media storage that fails the test if it is used.
func noMedia(t *testing.T) *mock.MockMediaStorage {
	t.Helper()
	return mock.NewMockMediaStorage(gomock.NewController(t))
}

// newMocks returns a repository and image storage that fail the test on any
// call they do not expect.
func newMocks(t *testing.T) (*mock.MockRepository, *mock.MockMediaStorage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mock.NewMockRepository(ctrl), mock.NewMockMediaStorage(ctrl)
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
			stored, err := createPlace(t, openCategory(), &places.Place{
				Name: tc.input, SubDistrictID: 1, CategoryID: 2,
			})
			if tc.wantErr {
				require.ErrorIs(t, err, places.ErrInvalidName)
				require.Nil(t, stored, "blank name reached storage")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, stored.Name)
		})
	}
}

func TestService_Create_NameTooLong(t *testing.T) {
	_, err := createPlace(t, openCategory(), &places.Place{
		Name: strings.Repeat("a", 151), SubDistrictID: 1, CategoryID: 2,
	})
	require.ErrorIs(t, err, places.ErrNameTooLong)
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
			stored, err := createPlace(t, warungCategory(), &places.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			if tc.wantErr {
				require.ErrorIs(t, err, places.ErrInvalidAttributes)
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
			stored, err := createPlace(t, openCategory(), &places.Place{
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

			var stored *places.Place
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, got *places.Place) (*places.Place, error) {
					stored = got
					return got, nil
				})

			_, err := places.NewService(repo, noMedia(t)).Create(context.Background(), &places.Place{
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

			_, err := places.NewService(repo, noMedia(t)).Create(context.Background(), &places.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			require.ErrorIs(t, err, places.ErrInvalidAttributes)
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
			p := &places.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
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
		{name: "unknown category stops before storage", lookupErr: places.ErrCategoryNotFound, storeErr: nil, wantErr: places.ErrCategoryNotFound},
		{name: "storage reports an unknown sub-district", lookupErr: nil, storeErr: places.ErrSubDistrictNotFound, wantErr: places.ErrSubDistrictNotFound},
		{name: "storage reports an unknown category", lookupErr: nil, storeErr: places.ErrCategoryNotFound, wantErr: places.ErrCategoryNotFound},
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
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&places.Place{ID: 7}, tc.storeErr)
			}

			got, err := places.NewService(repo, noMedia(t)).Create(context.Background(), &places.Place{
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

// savedKopi is the stored place that updates start from: a Kopi in Menteng
// with every optional field set.
func savedKopi() *places.Place {
	return &places.Place{
		ID:                  7,
		SubDistrictID:       1,
		CategoryID:          2,
		Name:                "Kopi Senja",
		Address:             new("Jl. Sudirman 9"),
		Location:            postgis.MustPoint(106.8, -6.2),
		OperatingHours:      new("08:00-17:00"),
		EstimatedPriceRange: new("Rp20.000-50.000"),
		ShortDescription:    new("Kopi tubruk"),
		Attributes:          json.RawMessage(`{"halal":true,"seats":12}`),
		Category:            places.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}
}

// lookupCategory answers category lookups for the update tests. Category 1 is
// Warung, which requires "contact". Category 2 is Kopi. Category 3 is Cafe.
func lookupCategory(_ context.Context, id int64) (*places.Category, error) {
	switch id {
	case 1:
		return warungCategory(), nil
	case 2:
		return openCategory(), nil
	case 3:
		return cafeCategory(), nil
	}
	return nil, places.ErrCategoryNotFound
}

// kopiInput is a full replacement for savedKopi: a Kopi in Menteng with every
// optional field set. Updates change one field of it at a time.
func kopiInput() *places.Place {
	return &places.Place{
		SubDistrictID:       1,
		CategoryID:          2,
		Name:                "Kopi Senja",
		Address:             new("Jl. Sudirman 9"),
		Location:            postgis.MustPoint(106.8, -6.2),
		OperatingHours:      new("08:00-17:00"),
		EstimatedPriceRange: new("Rp20.000-50.000"),
		ShortDescription:    new("Kopi tubruk"),
		Attributes:          json.RawMessage(`{"halal":true,"seats":12}`),
	}
}

// updatePlace runs Service.Update for place id with the replacement p. It
// returns the place the repository was asked to store, or nil if storage was
// never reached.
func updatePlace(t *testing.T, id int64, p *places.Place) (*places.Place, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetCategory(gomock.Any(), gomock.Any()).DoAndReturn(lookupCategory).AnyTimes()

	var stored *places.Place
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *places.Place) (*places.Place, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := places.NewService(repo, noMedia(t)).Update(context.Background(), id, p)
	return stored, err
}

func TestService_Update_Fields(t *testing.T) {
	tests := []struct {
		name   string
		change func(*places.Place) // turns kopiInput into the replacement
		want   func(*places.Place) // turns kopiInput into the wanted stored place
	}{
		{name: "same fields change nothing", change: func(*places.Place) {}, want: func(*places.Place) {}},
		{name: "name is trimmed",
			change: func(p *places.Place) { p.Name = "  Kopi Baru\t" },
			want:   func(p *places.Place) { p.Name = "Kopi Baru" }},
		{name: "address is replaced",
			change: func(p *places.Place) { p.Address = new("Jl. Kemang 2") },
			want:   func(p *places.Place) { p.Address = new("Jl. Kemang 2") }},
		{name: "location is replaced",
			change: func(p *places.Place) { p.Location = postgis.MustPoint(106.85, -6.18) },
			want:   func(p *places.Place) { p.Location = postgis.MustPoint(106.85, -6.18) }},
		{name: "sub-district is replaced",
			change: func(p *places.Place) { p.SubDistrictID = 2 },
			want:   func(p *places.Place) { p.SubDistrictID = 2 }},
		{name: "absent nullable fields are cleared",
			change: func(p *places.Place) {
				p.Address, p.Location, p.OperatingHours = nil, nil, nil
				p.EstimatedPriceRange, p.ShortDescription = nil, nil
			},
			want: func(p *places.Place) {
				p.Address, p.Location, p.OperatingHours = nil, nil, nil
				p.EstimatedPriceRange, p.ShortDescription = nil, nil
			}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := kopiInput()
			want.ID = 7
			tc.want(want)

			input := kopiInput()
			tc.change(input)

			stored, err := updatePlace(t, 7, input)
			require.NoError(t, err)
			assert.Equal(t, want, stored)
		})
	}
}

func TestService_Update_ID(t *testing.T) {
	input := kopiInput()
	input.ID = 99 // the place comes from the path, not from the replacement

	stored, err := updatePlace(t, 7, input)
	require.NoError(t, err)
	assert.Equal(t, int64(7), stored.ID)
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
		{name: "empty", input: "", wantName: "", wantErr: places.ErrInvalidName},
		{name: "spaces only", input: "   ", wantName: "", wantErr: places.ErrInvalidName},
		{name: "150 characters is allowed", input: long, wantName: long, wantErr: nil},
		{name: "151 characters is too long", input: long + "a", wantName: "", wantErr: places.ErrNameTooLong},
		{name: "150 multibyte characters is allowed", input: strings.Repeat("é", 150), wantName: strings.Repeat("é", 150), wantErr: nil},
		{name: "151 multibyte characters is too long", input: strings.Repeat("é", 151), wantName: "", wantErr: places.ErrNameTooLong},
		{name: "length is counted after trimming", input: "  " + long + "  ", wantName: long, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.Name = tc.input

			stored, err := updatePlace(t, 7, input)
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
		{name: "west of the Indonesia box", lon: 94.9, lat: -2, wantErr: places.ErrInvalidLocation},
		{name: "north of the Indonesia box", lon: 110, lat: 6.5, wantErr: places.ErrInvalidLocation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.Location = postgis.MustPoint(tc.lon, tc.lat)

			stored, err := updatePlace(t, 7, input)
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
		attrs     json.RawMessage
		wantAttrs string
		wantErr   error
	}{
		{name: "attributes are stored", attrs: json.RawMessage(`{"halal":false}`), wantAttrs: `{"halal":false}`, wantErr: nil},
		{name: "null attributes become an empty object", attrs: json.RawMessage(`null`), wantAttrs: `{}`, wantErr: nil},
		{name: "absent attributes become an empty object", attrs: nil, wantAttrs: `{}`, wantErr: nil},
		{name: "unknown key is rejected", attrs: json.RawMessage(`{"parking":true}`), wantAttrs: "", wantErr: places.ErrInvalidAttributes},
		{name: "wrong type is rejected", attrs: json.RawMessage(`{"seats":"many"}`), wantAttrs: "", wantErr: places.ErrInvalidAttributes},
		{name: "array is rejected", attrs: json.RawMessage(`[1]`), wantAttrs: "", wantErr: places.ErrInvalidAttributes},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.Attributes = tc.attrs

			stored, err := updatePlace(t, 7, input)
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
		attrs     string // attributes sent in the update
		wantAttrs string
	}{
		{name: "empty string is dropped", attrs: `{"website":"","seats":12}`, wantAttrs: `{"seats":12}`},
		{name: "spaces only is dropped", attrs: `{"website":"   ","seats":12}`, wantAttrs: `{"seats":12}`},
		{name: "tabs and newlines only is dropped", attrs: `{"website":"\t\n"}`, wantAttrs: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.CategoryID = 3 // Cafe
			input.Attributes = json.RawMessage(tc.attrs)

			stored, err := updatePlace(t, 7, input)
			require.NoError(t, err)
			assertJSON(t, stored.Attributes, tc.wantAttrs)
		})
	}
}

func TestService_Update_BlankRequiredText(t *testing.T) {
	for _, attrs := range []string{`{"contact":""}`, `{"contact":"   "}`, `{"contact":"\t\n"}`} {
		t.Run(attrs, func(t *testing.T) {
			// The place is a Warung, which requires "contact".
			input := kopiInput()
			input.CategoryID = 1
			input.Attributes = json.RawMessage(attrs)

			stored, err := updatePlace(t, 7, input)
			require.ErrorIs(t, err, places.ErrInvalidAttributes)
			require.Nil(t, stored, "blank required text reached storage")
		})
	}
}

func TestService_Update_Category(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     int64
		attrs          json.RawMessage
		wantCategoryID int64
		wantAttrs      string
		wantErr        error
	}{
		{
			name:           "attributes that fit the category are stored",
			categoryID:     1,
			attrs:          json.RawMessage(`{"contact":"0812","seats":12}`),
			wantCategoryID: 1,
			wantAttrs:      `{"contact":"0812","seats":12}`,
		},
		{
			name:       "attributes missing a required field of the category are rejected",
			categoryID: 1,
			attrs:      json.RawMessage(`{"halal":true,"seats":12}`),
			wantErr:    places.ErrInvalidAttributes,
		},
		{
			name:       "absent attributes are rejected when the category requires a field",
			categoryID: 1,
			attrs:      nil,
			wantErr:    places.ErrInvalidAttributes,
		},
		{
			name:       "attributes unknown to the category are rejected",
			categoryID: 2,
			attrs:      json.RawMessage(`{"contact":"0812"}`),
			wantErr:    places.ErrInvalidAttributes,
		},
		{
			name:       "unknown category is rejected",
			categoryID: 99,
			attrs:      json.RawMessage(`{"halal":true}`),
			wantErr:    places.ErrCategoryNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.CategoryID = tc.categoryID
			input.Attributes = tc.attrs

			stored, err := updatePlace(t, 7, input)
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
		{name: "storage reports the place is gone", updateErr: places.ErrPlaceNotFound, wantErr: places.ErrPlaceNotFound},
		{name: "storage reports an unknown sub-district", updateErr: places.ErrSubDistrictNotFound, wantErr: places.ErrSubDistrictNotFound},
		{name: "storage failure is returned", updateErr: boom, wantErr: boom},
		{name: "stored place is returned", updateErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetCategory(gomock.Any(), int64(2)).Return(openCategory(), nil)
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&places.Place{ID: 7}, tc.updateErr)

			got, err := places.NewService(repo, noMedia(t)).Update(context.Background(), 7, kopiInput())
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored place was not returned")
				assert.Equal(t, int64(7), got.ID, "want the stored place's ID")
			}
		})
	}
}

func TestService_GetMany_LimitDefaultsToAndIsCappedAtMaximum(t *testing.T) {
	const maximum = 100

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "absent limit gets the maximum", limit: 0, want: maximum},
		{name: "limit within the maximum is kept", limit: 25, want: 25},
		{name: "limit at the maximum is kept", limit: maximum, want: maximum},
		{name: "limit above the maximum is capped", limit: maximum + 1, want: maximum},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().GetMany(gomock.Any(), places.PlaceFilter{Query: "kopi", Limit: tc.want}).
				Return([]places.Place{}, 0, nil)

			_, _, err := places.NewService(repo, media).GetMany(context.Background(),
				places.PlaceFilter{Query: "kopi", Limit: tc.limit})
			assert.NoError(t, err, "get many")
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
		{name: "unknown place", repoErr: places.ErrPlaceNotFound, wantErr: places.ErrPlaceNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues any medias, so no image may be deleted.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := places.NewService(repo, noMedia(t)).Delete(context.Background(), 7)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_Delete_Medias(t *testing.T) {
	// The repository queues the medias for deletion, so the service leaves
	// them to the cleanup worker and must not delete them itself.
	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil)

	err := places.NewService(repo, noMedia(t)).Delete(context.Background(), 7)
	require.NoError(t, err, "the place was deleted")
}

// -----------------------------------------------------------------------------
// Optional text fields
// -----------------------------------------------------------------------------

// placeTextField is an optional text field of a place. limit is the length its
// column allows in characters, or 0 when the field has no limit.
type placeTextField struct {
	name    string
	limit   int
	tooLong error
	set     func(p *places.Place, v *string)
	get     func(p *places.Place) *string
}

var placeTextFields = []placeTextField{
	{name: "address", limit: places.MaxAddressLength, tooLong: places.ErrAddressTooLong,
		set: func(p *places.Place, v *string) { p.Address = v },
		get: func(p *places.Place) *string { return p.Address }},
	{name: "operating hours", limit: places.MaxOperatingHoursLength, tooLong: places.ErrOperatingHoursTooLong,
		set: func(p *places.Place, v *string) { p.OperatingHours = v },
		get: func(p *places.Place) *string { return p.OperatingHours }},
	{name: "estimated price range", limit: places.MaxEstimatedPriceRangeLength, tooLong: places.ErrEstimatedPriceRangeTooLong,
		set: func(p *places.Place, v *string) { p.EstimatedPriceRange = v },
		get: func(p *places.Place) *string { return p.EstimatedPriceRange }},
	{name: "short description",
		set: func(p *places.Place, v *string) { p.ShortDescription = v },
		get: func(p *places.Place) *string { return p.ShortDescription }},
}

func TestService_Create_OptionalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{name: "surrounding whitespace is removed", input: "  Jl. Sudirman  ", want: new("Jl. Sudirman")},
		{name: "blank is stored as absent", input: " \t\n ", want: nil},
	}

	for _, f := range placeTextFields {
		for _, tc := range tests {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				p := &places.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
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
				p := &places.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
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
			{name: "surrounding whitespace is removed", input: "  Jl. Sudirman  ", want: new("Jl. Sudirman")},
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
				input := kopiInput()
				value := tc.input
				f.set(input, &value)

				stored, err := updatePlace(t, 7, input)
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
// Place medias
// -----------------------------------------------------------------------------

const (
	uploadKey = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	imageKey  = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
)

// promoted is the media Promote returns for uploadKey.
var promoted = storage.Media{Key: imageKey, ContentType: "image/webp", Size: 2048}

func TestService_MediaUploadURL(t *testing.T) {
	repo, media := newMocks(t)
	upload := storage.Upload{
		Key:    uploadKey,
		URL:    "http://localhost:3900/signed",
		Method: "PUT",
	}
	gomock.InOrder(
		repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil),
		media.EXPECT().PresignUpload(gomock.Any(), "places/7", "image/webp").Return(upload, nil),
	)

	got, err := places.NewService(repo, media).MediaUploadURL(context.Background(), 7, "image/webp")
	require.NoError(t, err)
	assert.Equal(t, upload, got)
}

func TestService_MediaUploadURL_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown place", getErr: places.ErrPlaceNotFound, wantErr: places.ErrPlaceNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be presigned for a place that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := places.NewService(repo, noMedia(t)).MediaUploadURL(context.Background(), 7, "image/webp")
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_MediaUploadURL_PresignFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		presignErr error
	}{
		{name: "invalid image type", presignErr: storage.ErrInvalidMediaType},
		{name: "storage failure is returned", presignErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil)
			media.EXPECT().PresignUpload(gomock.Any(), "places/7", "image/gif").Return(storage.Upload{}, tc.presignErr)

			_, err := places.NewService(repo, media).MediaUploadURL(context.Background(), 7, "image/gif")
			require.ErrorIs(t, err, tc.presignErr)
		})
	}
}

func TestService_AddMedia(t *testing.T) {
	// The promoted media is stored as it is, and nothing is deleted.
	repo, media := newMocks(t)
	updated := savedKopi()
	updated.Medias = []storage.Media{{ID: 3, Key: imageKey, ContentType: "image/webp", Size: 2048, Position: 1}}

	gomock.InOrder(
		repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil),
		media.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(promoted, nil),
		repo.EXPECT().AddMedia(gomock.Any(), int64(7), promoted).Return(updated, nil),
	)

	got, err := places.NewService(repo, media).AddMedia(context.Background(), 7, uploadKey)
	require.NoError(t, err)
	assert.Same(t, updated, got, "want the stored place")
}

func TestService_AddMedia_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown place", getErr: places.ErrPlaceNotFound, wantErr: places.ErrPlaceNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be promoted for a place that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := places.NewService(repo, noMedia(t)).AddMedia(context.Background(), 7, uploadKey)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_AddMedia_PromoteFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		promoteErr error
	}{
		{name: "invalid upload key", promoteErr: storage.ErrInvalidUploadKey},
		{name: "image not uploaded", promoteErr: storage.ErrMediaNotUploaded},
		{name: "invalid image type", promoteErr: storage.ErrInvalidMediaType},
		{name: "image too large", promoteErr: storage.ErrImageTooLarge},
		{name: "storage failure is returned", promoteErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be stored.
			repo, media := newMocks(t)
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil)
			media.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(storage.Media{}, tc.promoteErr)

			_, err := places.NewService(repo, media).AddMedia(context.Background(), 7, uploadKey)
			require.ErrorIs(t, err, tc.promoteErr)
		})
	}
}

func TestService_AddMedia_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		addErr    error
		deleteErr error
	}{
		{name: "place deleted meanwhile", addErr: places.ErrPlaceNotFound},
		{name: "place already has the most medias", addErr: storage.ErrMediaLimitReached},
		{name: "storage failure is returned", addErr: boom},
		{name: "promoted image delete failure is ignored", addErr: boom, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The promoted image is removed.
			repo, media := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil),
				media.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(promoted, nil),
				repo.EXPECT().AddMedia(gomock.Any(), int64(7), promoted).Return(nil, tc.addErr),
				media.EXPECT().Delete(gomock.Any(), imageKey).Return(tc.deleteErr),
			)

			_, err := places.NewService(repo, media).AddMedia(context.Background(), 7, uploadKey)
			require.ErrorIs(t, err, tc.addErr)
		})
	}
}

func TestService_DeleteMedia(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "deletes the media", repoErr: nil},
		{name: "unknown media", repoErr: storage.ErrMediaNotFound},
		{name: "storage failure is returned", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues the image for deletion, so the service
			// must not delete it itself.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().DeleteMedia(gomock.Any(), int64(7), int64(3)).Return(tc.repoErr)

			err := places.NewService(repo, noMedia(t)).DeleteMedia(context.Background(), 7, 3)
			require.ErrorIs(t, err, tc.repoErr)
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
		{name: "unknown place", repoErr: places.ErrPlaceNotFound, wantErr: places.ErrPlaceNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), tc.repoErr)

			got, err := places.NewService(repo, noMedia(t)).GetPlace(context.Background(), 7)
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
		{name: "unknown category", repoErr: places.ErrCategoryNotFound, wantErr: places.ErrCategoryNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetCategory(gomock.Any(), int64(1)).Return(warungCategory(), tc.repoErr)

			got, err := places.NewService(repo, noMedia(t)).GetCategory(context.Background(), 1)
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored category was not returned")
				assert.Equal(t, int64(1), got.ID)
				assert.Len(t, got.Fields, 6, "Warung has 6 fields")
			}
		})
	}
}
