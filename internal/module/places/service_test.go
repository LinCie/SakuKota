package places_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/module/places/mock"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/storage"
)

var errBoom = errors.New("connection reset")

// warungCategory has one required field and one optional field of each data
// type that attributes can hold.
func warungCategory() *domain.Category {
	return &domain.Category{
		ID:   1,
		Name: "Warung",
		Slug: "warung",
		Fields: []domain.CategoryField{
			{Key: "contact", Label: "Contact", DataType: domain.FieldTypeText, Required: true},
			{Key: "seats", Label: "Seats", DataType: domain.FieldTypeNumber},
			{Key: "halal", Label: "Halal", DataType: domain.FieldTypeBoolean},
			{Key: "payment", Label: "Payment", DataType: domain.FieldTypeSelect, Options: domain.FieldOptions{"cash", "card"}},
			{Key: "price_range", Label: "Price range", DataType: domain.FieldTypeRange},
			{Key: "opened_on", Label: "Opened on", DataType: domain.FieldTypeDate},
		},
	}
}

// openCategory has no required fields.
func openCategory() *domain.Category {
	return &domain.Category{
		ID:   2,
		Name: "Kopi",
		Slug: "kopi",
		Fields: []domain.CategoryField{
			{Key: "seats", Label: "Seats", DataType: domain.FieldTypeNumber},
			{Key: "halal", Label: "Halal", DataType: domain.FieldTypeBoolean},
		},
	}
}

// cafeCategory has no required fields. Its optional text field "website" is
// the text attribute the blank-as-absent rule applies to.
func cafeCategory() *domain.Category {
	return &domain.Category{
		ID:   3,
		Name: "Cafe",
		Slug: "cafe",
		Fields: []domain.CategoryField{
			{Key: "website", Label: "Website", DataType: domain.FieldTypeText},
			{Key: "seats", Label: "Seats", DataType: domain.FieldTypeNumber},
		},
	}
}

// newMocks returns a mock repository and media storage that fail the test when
// it ends if an expected call did not happen, or when an unexpected one does.
func newMocks(t *testing.T) (*mock.MockRepository, *mock.MockMediaStorage) {
	t.Helper()

	ctrl := gomock.NewController(t)
	return mock.NewMockRepository(ctrl), mock.NewMockMediaStorage(ctrl)
}

// createPlace runs Service.Create for p in category c. It returns the place the
// repository was asked to store, or nil if storage was never reached.
func createPlace(t *testing.T, c *domain.Category, p *domain.Place) (*domain.Place, error) {
	t.Helper()

	repo, media := newMocks(t)
	repo.EXPECT().FetchCategory(gomock.Any(), p.CategoryID).Return(c, nil).AnyTimes()

	var stored *domain.Place
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *domain.Place) (*domain.Place, error) {
		stored = got
		return got, nil
	}).AnyTimes()

	_, err := places.NewService(repo, media).Create(t.Context(), p)
	return stored, err
}

// decodeExact decodes the JSON document raw. Numbers keep their literal text,
// so 1.50 and 1.5 decode to different values.
func decodeExact(t *testing.T, raw json.RawMessage) any {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var v any
	require.NoError(t, dec.Decode(&v), "invalid JSON %s", raw)

	return v
}

func TestService_Create_Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  bool
	}{
		{name: "trim_spaces", input: "  Warung Bu Ani  ", wantName: "Warung Bu Ani", wantErr: false},
		{name: "trim_tabs_newlines", input: "\tWarung Bu Ani\n", wantName: "Warung Bu Ani", wantErr: false},
		{name: "keep_inner_spacing", input: "Warung  Bu Ani", wantName: "Warung  Bu Ani", wantErr: false},
		{name: "empty", input: "", wantName: "", wantErr: true},
		{name: "spaces_only", input: "   ", wantName: "", wantErr: true},
		{name: "tabs_newlines_only", input: "\t\n", wantName: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, openCategory(), &domain.Place{
				Name: tc.input, SubDistrictID: 1, CategoryID: 2,
			})
			if tc.wantErr {
				assert.ErrorIs(t, err, domain.ErrPlaceNameRequired, "Create(name %q)", tc.input)
				assert.Nil(t, stored, "Create(name %q) must keep a blank name out of storage", tc.input)
				return
			}
			require.NoError(t, err, "Create(name %q)", tc.input)
			assert.Equal(t, tc.wantName, stored.Name, "Create(name %q) stored name", tc.input)
		})
	}
}

func TestService_Create_NameTooLong(t *testing.T) {
	_, err := createPlace(t, openCategory(), &domain.Place{
		Name: strings.Repeat("a", 151), SubDistrictID: 1, CategoryID: 2,
	})
	assert.ErrorIs(t, err, domain.ErrPlaceNameTooLong, "Create(151-character name)")
}

func TestService_Create_Attributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string // stored JSON when the attributes are accepted
		wantErr    bool
	}{
		{name: "required_only", attributes: `{"contact":"0812-3456"}`, wantStored: `{"contact":"0812-3456"}`, wantErr: false},
		{
			name:       "every_type_valid",
			attributes: `{"contact":"0812","seats":12,"halal":true,"payment":"card","price_range":{"min":10000,"max":50000},"opened_on":"2024-01-31"}`,
			wantStored: `{"contact":"0812","seats":12,"halal":true,"payment":"card","price_range":{"min":10000,"max":50000},"opened_on":"2024-01-31"}`,
			wantErr:    false,
		},
		{name: "range_equal_bounds", attributes: `{"contact":"x","price_range":{"min":5,"max":5}}`, wantStored: `{"contact":"x","price_range":{"min":5,"max":5}}`, wantErr: false},
		{name: "integer_keeps_digits", attributes: `{"contact":"x","seats":12345678901234567890}`, wantStored: `{"contact":"x","seats":12345678901234567890}`, wantErr: false},
		{name: "decimal_keeps_zero", attributes: `{"contact":"x","seats":1.50}`, wantStored: `{"contact":"x","seats":1.50}`, wantErr: false},
		{
			name:       "text_verbatim",
			attributes: `{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`,
			wantStored: `{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`,
			wantErr:    false,
		},
		{name: "optional_null_dropped", attributes: `{"contact":"x","seats":null,"halal":null}`, wantStored: `{"contact":"x"}`, wantErr: false},

		{name: "required_missing", attributes: `{"seats":1}`, wantStored: "", wantErr: true},
		{name: "required_spaces", attributes: `{"contact":"   "}`, wantStored: "", wantErr: true},
		{name: "required_null", attributes: `{"contact":null}`, wantStored: "", wantErr: true},
		{name: "null_attributes", attributes: `null`, wantStored: "", wantErr: true},
		{name: "empty_object", attributes: `{}`, wantStored: "", wantErr: true},
		{name: "array", attributes: `[1,2]`, wantStored: "", wantErr: true},
		{name: "string", attributes: `"text"`, wantStored: "", wantErr: true},
		{name: "malformed", attributes: `{"contact":`, wantStored: "", wantErr: true},
		{name: "unknown_key", attributes: `{"contact":"x","parking":true}`, wantStored: "", wantErr: true},
		{name: "text_number", attributes: `{"contact":5}`, wantStored: "", wantErr: true},
		{name: "number_string", attributes: `{"contact":"x","seats":"12"}`, wantStored: "", wantErr: true},
		{name: "number_boolean", attributes: `{"contact":"x","seats":true}`, wantStored: "", wantErr: true},
		{name: "boolean_string", attributes: `{"contact":"x","halal":"true"}`, wantStored: "", wantErr: true},
		{name: "boolean_number", attributes: `{"contact":"x","halal":1}`, wantStored: "", wantErr: true},
		{name: "select_outside_options", attributes: `{"contact":"x","payment":"bitcoin"}`, wantStored: "", wantErr: true},
		{name: "select_number", attributes: `{"contact":"x","payment":1}`, wantStored: "", wantErr: true},
		{name: "range_min_above_max", attributes: `{"contact":"x","price_range":{"min":9,"max":1}}`, wantStored: "", wantErr: true},
		{name: "range_missing_max", attributes: `{"contact":"x","price_range":{"min":1}}`, wantStored: "", wantErr: true},
		{name: "range_extra_key", attributes: `{"contact":"x","price_range":{"min":1,"max":2,"step":1}}`, wantStored: "", wantErr: true},
		{name: "range_string_bounds", attributes: `{"contact":"x","price_range":{"min":"1","max":"2"}}`, wantStored: "", wantErr: true},
		{name: "range_array", attributes: `{"contact":"x","price_range":[1,2]}`, wantStored: "", wantErr: true},
		{name: "date_slashes", attributes: `{"contact":"x","opened_on":"2024/01/31"}`, wantStored: "", wantErr: true},
		{name: "date_nonexistent", attributes: `{"contact":"x","opened_on":"2024-02-30"}`, wantStored: "", wantErr: true},
		{name: "date_number", attributes: `{"contact":"x","opened_on":20240131}`, wantStored: "", wantErr: true},
		{name: "date_with_time", attributes: `{"contact":"x","opened_on":"2024-01-31T00:00:00Z"}`, wantStored: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, warungCategory(), &domain.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			if tc.wantErr {
				assert.ErrorIs(t, err, domain.ErrPlaceAttributesInvalid, "Create(attributes %s)", tc.attributes)
				assert.Nil(t, stored, "Create(attributes %s) must keep rejected attributes out of storage", tc.attributes)
				return
			}
			require.NoError(t, err, "Create(attributes %s)", tc.attributes)
			assert.Equal(t, decodeExact(t, json.RawMessage(tc.wantStored)), decodeExact(t, stored.Attributes), "Create(attributes %s) stored attributes", tc.attributes)
		})
	}
}

func TestService_Create_AttributesWithoutRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string
	}{
		{name: "absent", attributes: "", wantStored: `{}`},
		{name: "null", attributes: `null`, wantStored: `{}`},
		{name: "empty_object", attributes: `{}`, wantStored: `{}`},
		{name: "optional_set", attributes: `{"seats":5}`, wantStored: `{"seats":5}`},
		{name: "optional_null", attributes: `{"halal":null}`, wantStored: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, openCategory(), &domain.Place{
				Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2,
				Attributes: json.RawMessage(tc.attributes),
			})
			require.NoError(t, err, "Create(attributes %q)", tc.attributes)
			assert.Equal(t, decodeExact(t, json.RawMessage(tc.wantStored)), decodeExact(t, stored.Attributes), "Create(attributes %q) stored attributes", tc.attributes)
		})
	}
}

func TestService_Create_BlankOptionalText(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string
	}{
		{name: "empty_dropped", attributes: `{"website":""}`, wantStored: `{}`},
		{name: "spaces_dropped", attributes: `{"website":"   "}`, wantStored: `{}`},
		{name: "tabs_newlines_dropped", attributes: `{"website":"\t\n"}`, wantStored: `{}`},
		{name: "other_keys_kept", attributes: `{"website":"","seats":5}`, wantStored: `{"seats":5}`},
		{name: "non_blank_verbatim", attributes: `{"website":"  Halal  "}`, wantStored: `{"website":"  Halal  "}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, cafeCategory(), &domain.Place{
				Name: "Kafe Senja", SubDistrictID: 1, CategoryID: 3,
				Attributes: json.RawMessage(tc.attributes),
			})
			require.NoError(t, err, "Create(attributes %s)", tc.attributes)
			require.NotNil(t, stored, "Create(attributes %s) must reach storage", tc.attributes)
			assert.Equal(t, decodeExact(t, json.RawMessage(tc.wantStored)), decodeExact(t, stored.Attributes), "Create(attributes %s) stored attributes", tc.attributes)
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
		{name: "required_empty", attributes: `{"contact":""}`},
		{name: "required_whitespace", attributes: `{"contact":" \t\n "}`},
		{name: "number_empty", attributes: `{"contact":"x","seats":""}`},
		{name: "boolean_empty", attributes: `{"contact":"x","halal":""}`},
		{name: "select_empty", attributes: `{"contact":"x","payment":""}`},
		{name: "range_empty", attributes: `{"contact":"x","price_range":""}`},
		{name: "date_empty", attributes: `{"contact":"x","opened_on":""}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// createPlace's repository accepts a Create, so a stored place means it was not rejected.
			stored, err := createPlace(t, warungCategory(), &domain.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			assert.ErrorIs(t, err, domain.ErrPlaceAttributesInvalid, "Create(attributes %s)", tc.attributes)
			assert.Nil(t, stored, "Create(attributes %s) must store nothing", tc.attributes)
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
		{name: "jakarta", lon: 106.8, lat: -6.2, omit: false, wantErr: false},
		{name: "sabang_west_tip", lon: 95.3, lat: 5.9, omit: false, wantErr: false},
		{name: "merauke_east_tip", lon: 140.4, lat: -8.5, omit: false, wantErr: false},
		{name: "west_edge", lon: 95, lat: -6.2, omit: false, wantErr: false},
		{name: "east_edge", lon: 141, lat: -6.2, omit: false, wantErr: false},
		{name: "north_edge", lon: 106.8, lat: 6, omit: false, wantErr: false},
		{name: "south_edge", lon: 106.8, lat: -11, omit: false, wantErr: false},
		{name: "omitted", lon: 0, lat: 0, omit: true, wantErr: false},
		{name: "tokyo", lon: 139.7, lat: 35.7, omit: false, wantErr: true},
		{name: "london", lon: -0.1, lat: 51.5, omit: false, wantErr: true},
		{name: "null_island", lon: 0, lat: 0, omit: false, wantErr: true},
		{name: "just_west", lon: 94.99, lat: -6.2, omit: false, wantErr: true},
		{name: "just_east", lon: 141.01, lat: -6.2, omit: false, wantErr: true},
		{name: "just_north", lon: 106.8, lat: 6.01, omit: false, wantErr: true},
		{name: "just_south", lon: 106.8, lat: -11.01, omit: false, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &domain.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
			if !tc.omit {
				point, err := postgis.NewPoint(tc.lon, tc.lat)
				require.NoError(t, err, "NewPoint(%v, %v)", tc.lon, tc.lat)
				p.Location = point
			}

			stored, err := createPlace(t, openCategory(), p)
			if tc.wantErr {
				assert.Error(t, err, "Create(location %v, %v) must reject a location outside Indonesia", tc.lon, tc.lat)
				assert.Nil(t, stored, "Create(location %v, %v) must store nothing", tc.lon, tc.lat)
				return
			}
			require.NoError(t, err, "Create(location %v, %v)", tc.lon, tc.lat)
			if tc.omit {
				assert.Nil(t, stored.Location, "Create(no location) stored location")
				return
			}
			require.NotNil(t, stored.Location, "Create(location %v, %v) stored location", tc.lon, tc.lat)
			assert.Equal(t, tc.lon, stored.Location.X(), "Create(location %v, %v) stored longitude", tc.lon, tc.lat)
			assert.Equal(t, tc.lat, stored.Location.Y(), "Create(location %v, %v) stored latitude", tc.lon, tc.lat)
		})
	}
}

func TestService_Create_Storage(t *testing.T) {
	tests := []struct {
		name      string
		lookupErr error // returned when the category is looked up
		storeErr  error // returned when the place is stored
		wantErr   error // nil means the call succeeds
	}{
		{name: "unknown_category", lookupErr: domain.ErrCategoryNotFound, storeErr: nil, wantErr: domain.ErrCategoryNotFound},
		{name: "storage_unknown_sub_district", lookupErr: nil, storeErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "storage_unknown_category", lookupErr: nil, storeErr: domain.ErrCategoryNotFound, wantErr: domain.ErrCategoryNotFound},
		{name: "storage_failure", lookupErr: nil, storeErr: errBoom, wantErr: errBoom},
		{name: "stored", lookupErr: nil, storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			if tc.lookupErr != nil {
				repo.EXPECT().FetchCategory(gomock.Any(), int64(2)).Return(nil, tc.lookupErr)
			} else {
				repo.EXPECT().FetchCategory(gomock.Any(), int64(2)).Return(openCategory(), nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&domain.Place{ID: 7}, tc.storeErr)
			}

			got, err := places.NewService(repo, media).Create(t.Context(), &domain.Place{
				Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2,
			})
			assert.ErrorIs(t, err, tc.wantErr, "Create()")
			if tc.wantErr == nil {
				require.NotNil(t, got, "Create() must return the stored place")
				assert.Equal(t, int64(7), got.ID, "Create().ID")
			}
		})
	}
}

// savedKopi is the stored place that updates start from: a Kopi in Menteng
// with every optional field set.
func savedKopi() *domain.Place {
	return &domain.Place{
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
		Category:            domain.Category{ID: 2, Name: "Kopi", Slug: "kopi", Code: "K", Label: "Kopi", Color: "plum"},
	}
}

// lookupCategory answers category lookups for the update tests. Category 1 is
// Warung, which requires "contact". Category 2 is Kopi. Category 3 is Cafe.
func lookupCategory(_ context.Context, id int64) (*domain.Category, error) {
	switch id {
	case 1:
		return warungCategory(), nil
	case 2:
		return openCategory(), nil
	case 3:
		return cafeCategory(), nil
	}
	return nil, domain.ErrCategoryNotFound
}

// kopiInput is a full replacement for savedKopi: a Kopi in Menteng with every
// optional field set. Updates change one field of it at a time.
func kopiInput() *domain.Place {
	return &domain.Place{
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
func updatePlace(t *testing.T, id int64, p *domain.Place) (*domain.Place, error) {
	t.Helper()

	repo, media := newMocks(t)
	repo.EXPECT().FetchCategory(gomock.Any(), p.CategoryID).DoAndReturn(lookupCategory).AnyTimes()

	var stored *domain.Place
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *domain.Place) (*domain.Place, error) {
		stored = got
		return got, nil
	}).AnyTimes()

	_, err := places.NewService(repo, media).Update(t.Context(), id, p)
	return stored, err
}

func TestService_Update_Fields(t *testing.T) {
	tests := []struct {
		name   string
		change func(*domain.Place) // turns kopiInput into the replacement
		want   func(*domain.Place) // turns kopiInput into the wanted stored place
	}{
		{name: "unchanged", change: func(*domain.Place) {}, want: func(*domain.Place) {}},
		{name: "name_trimmed",
			change: func(p *domain.Place) { p.Name = "  Kopi Baru\t" },
			want:   func(p *domain.Place) { p.Name = "Kopi Baru" }},
		{name: "address_replaced",
			change: func(p *domain.Place) { p.Address = new("Jl. Kemang 2") },
			want:   func(p *domain.Place) { p.Address = new("Jl. Kemang 2") }},
		{name: "location_replaced",
			change: func(p *domain.Place) { p.Location = postgis.MustPoint(106.85, -6.18) },
			want:   func(p *domain.Place) { p.Location = postgis.MustPoint(106.85, -6.18) }},
		{name: "sub_district_replaced",
			change: func(p *domain.Place) { p.SubDistrictID = 2 },
			want:   func(p *domain.Place) { p.SubDistrictID = 2 }},
		{name: "nullable_cleared",
			change: func(p *domain.Place) {
				p.Address, p.Location, p.OperatingHours = nil, nil, nil
				p.EstimatedPriceRange, p.ShortDescription = nil, nil
			},
			want: func(p *domain.Place) {
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
			require.NoError(t, err, "Update(7, %+v)", input)
			assert.Equal(t, want, stored, "Update(7, %+v) stored place", input)
		})
	}
}

func TestService_Update_ID(t *testing.T) {
	input := kopiInput()
	input.ID = 99 // the place comes from the path, not from the replacement

	stored, err := updatePlace(t, 7, input)
	require.NoError(t, err, "Update(7, ID 99)")
	assert.Equal(t, int64(7), stored.ID, "Update(7, ID 99) stored ID")
}

func TestService_Update_Name(t *testing.T) {
	long := strings.Repeat("a", 150)

	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  error
	}{
		{name: "trim_spaces", input: "  Kopi Baru  ", wantName: "Kopi Baru", wantErr: nil},
		{name: "empty", input: "", wantName: "", wantErr: domain.ErrPlaceNameRequired},
		{name: "spaces_only", input: "   ", wantName: "", wantErr: domain.ErrPlaceNameRequired},
		{name: "150_chars", input: long, wantName: long, wantErr: nil},
		{name: "151_chars", input: long + "a", wantName: "", wantErr: domain.ErrPlaceNameTooLong},
		{name: "150_multibyte", input: strings.Repeat("é", 150), wantName: strings.Repeat("é", 150), wantErr: nil},
		{name: "151_multibyte", input: strings.Repeat("é", 151), wantName: "", wantErr: domain.ErrPlaceNameTooLong},
		{name: "counted_after_trim", input: "  " + long + "  ", wantName: long, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.Name = tc.input

			stored, err := updatePlace(t, 7, input)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Update(7, name %q)", tc.input)
				assert.Nil(t, stored, "Update(7, name %q) must keep an invalid name out of storage", tc.input)
				return
			}
			require.NoError(t, err, "Update(7, name %q)", tc.input)
			assert.Equal(t, tc.wantName, stored.Name, "Update(7, name %q) stored name", tc.input)
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
		{name: "inside", lon: 106.8, lat: -6.2, wantErr: nil},
		{name: "west_edge", lon: 95, lat: -2, wantErr: nil},
		{name: "west_outside", lon: 94.9, lat: -2, wantErr: domain.ErrPlaceLocationInvalid},
		{name: "north_outside", lon: 110, lat: 6.5, wantErr: domain.ErrPlaceLocationInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.Location = postgis.MustPoint(tc.lon, tc.lat)

			stored, err := updatePlace(t, 7, input)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Update(7, location %v, %v)", tc.lon, tc.lat)
				assert.Nil(t, stored, "Update(7, location %v, %v) must store nothing", tc.lon, tc.lat)
				return
			}
			require.NoError(t, err, "Update(7, location %v, %v)", tc.lon, tc.lat)
			require.NotNil(t, stored.Location, "Update(7, location %v, %v) stored location", tc.lon, tc.lat)
			assert.Equal(t, tc.lon, stored.Location.X(), "Update(7, location %v, %v) stored longitude", tc.lon, tc.lat)
			assert.Equal(t, tc.lat, stored.Location.Y(), "Update(7, location %v, %v) stored latitude", tc.lon, tc.lat)
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
		{name: "stored", attrs: json.RawMessage(`{"halal":false}`), wantAttrs: `{"halal":false}`, wantErr: nil},
		{name: "null", attrs: json.RawMessage(`null`), wantAttrs: `{}`, wantErr: nil},
		{name: "absent", attrs: nil, wantAttrs: `{}`, wantErr: nil},
		{name: "unknown_key", attrs: json.RawMessage(`{"parking":true}`), wantAttrs: "", wantErr: domain.ErrPlaceAttributesInvalid},
		{name: "wrong_type", attrs: json.RawMessage(`{"seats":"many"}`), wantAttrs: "", wantErr: domain.ErrPlaceAttributesInvalid},
		{name: "array", attrs: json.RawMessage(`[1]`), wantAttrs: "", wantErr: domain.ErrPlaceAttributesInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.Attributes = tc.attrs

			stored, err := updatePlace(t, 7, input)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Update(7, attributes %s)", tc.attrs)
				assert.Nil(t, stored, "Update(7, attributes %s) must keep invalid attributes out of storage", tc.attrs)
				return
			}
			require.NoError(t, err, "Update(7, attributes %s)", tc.attrs)
			assert.Equal(t, decodeExact(t, json.RawMessage(tc.wantAttrs)), decodeExact(t, stored.Attributes), "Update(7, attributes %s) stored attributes", tc.attrs)
		})
	}
}

func TestService_Update_BlankOptionalText(t *testing.T) {
	tests := []struct {
		name      string
		attrs     string // attributes sent in the update
		wantAttrs string
	}{
		{name: "empty_dropped", attrs: `{"website":"","seats":12}`, wantAttrs: `{"seats":12}`},
		{name: "spaces_dropped", attrs: `{"website":"   ","seats":12}`, wantAttrs: `{"seats":12}`},
		{name: "tabs_newlines_dropped", attrs: `{"website":"\t\n"}`, wantAttrs: `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.CategoryID = 3 // Cafe
			input.Attributes = json.RawMessage(tc.attrs)

			stored, err := updatePlace(t, 7, input)
			require.NoError(t, err, "Update(7, attributes %s)", tc.attrs)
			assert.Equal(t, decodeExact(t, json.RawMessage(tc.wantAttrs)), decodeExact(t, stored.Attributes), "Update(7, attributes %s) stored attributes", tc.attrs)
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
			assert.ErrorIs(t, err, domain.ErrPlaceAttributesInvalid, "Update(7, attributes %s)", attrs)
			assert.Nil(t, stored, "Update(7, attributes %s) must keep blank required text out of storage", attrs)
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
			name:           "fits_category",
			categoryID:     1,
			attrs:          json.RawMessage(`{"contact":"0812","seats":12}`),
			wantCategoryID: 1,
			wantAttrs:      `{"contact":"0812","seats":12}`,
		},
		{
			name:       "missing_required",
			categoryID: 1,
			attrs:      json.RawMessage(`{"halal":true,"seats":12}`),
			wantErr:    domain.ErrPlaceAttributesInvalid,
		},
		{
			name:       "absent_but_required",
			categoryID: 1,
			attrs:      nil,
			wantErr:    domain.ErrPlaceAttributesInvalid,
		},
		{
			name:       "unknown_to_category",
			categoryID: 2,
			attrs:      json.RawMessage(`{"contact":"0812"}`),
			wantErr:    domain.ErrPlaceAttributesInvalid,
		},
		{
			name:       "unknown_category",
			categoryID: 99,
			attrs:      json.RawMessage(`{"halal":true}`),
			wantErr:    domain.ErrCategoryNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := kopiInput()
			input.CategoryID = tc.categoryID
			input.Attributes = tc.attrs

			stored, err := updatePlace(t, 7, input)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Update(7, category %d)", tc.categoryID)
				assert.Nil(t, stored, "Update(7, category %d) must keep a rejected update out of storage", tc.categoryID)
				return
			}
			require.NoError(t, err, "Update(7, category %d)", tc.categoryID)
			assert.Equal(t, tc.wantCategoryID, stored.CategoryID, "Update(7, category %d) stored category", tc.categoryID)
			assert.Equal(t, decodeExact(t, json.RawMessage(tc.wantAttrs)), decodeExact(t, stored.Attributes), "Update(7, category %d) stored attributes", tc.categoryID)
		})
	}
}

func TestService_Update_Storage(t *testing.T) {
	tests := []struct {
		name      string
		updateErr error // returned when the place is stored
		wantErr   error // nil means the call succeeds
	}{
		{name: "place_gone", updateErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "unknown_sub_district", updateErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "storage_failure", updateErr: errBoom, wantErr: errBoom},
		{name: "stored", updateErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().FetchCategory(gomock.Any(), int64(2)).Return(openCategory(), nil)
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&domain.Place{ID: 7}, tc.updateErr)

			got, err := places.NewService(repo, media).Update(t.Context(), 7, kopiInput())
			assert.ErrorIs(t, err, tc.wantErr, "Update(7)")
			if tc.wantErr == nil {
				require.NotNil(t, got, "Update(7) must return the stored place")
				assert.Equal(t, int64(7), got.ID, "Update(7).ID")
			}
		})
	}
}

func TestService_List_LimitDefaultsToAndIsCappedAtMaximum(t *testing.T) {
	const maximum = 100

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "absent", limit: 0, want: maximum},
		{name: "within_maximum", limit: 25, want: 25},
		{name: "at_maximum", limit: maximum, want: maximum},
		{name: "above_maximum", limit: maximum + 1, want: maximum},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().List(gomock.Any(), places.Filter{Query: "kopi", Limit: tc.want}).Return([]domain.Place{}, 0, nil)

			_, _, err := places.NewService(repo, media).List(t.Context(), places.Filter{Query: "kopi", Limit: tc.limit})
			assert.NoError(t, err, "List(limit %d)", tc.limit)
		})
	}
}

func TestService_Delete(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "deleted", repoErr: nil, wantErr: nil},
		{name: "unknown_place", repoErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "storage_failure", repoErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues any medias, so no image may be deleted.
			repo, media := newMocks(t)
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := places.NewService(repo, media).Delete(t.Context(), 7)
			assert.ErrorIs(t, err, tc.wantErr, "Delete(7)")
		})
	}
}

func TestService_Delete_Medias(t *testing.T) {
	// The repository queues the medias for deletion, so the service leaves
	// them to the cleanup worker and must not delete them itself.
	repo, media := newMocks(t)
	repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil)

	assert.NoError(t, places.NewService(repo, media).Delete(t.Context(), 7), "Delete(7)")
}

// placeTextField is an optional text field of a place. limit is the length its
// column allows in characters, or 0 when the field has no limit.
type placeTextField struct {
	name    string
	limit   int
	tooLong error
	set     func(p *domain.Place, v *string)
	get     func(p *domain.Place) *string
}

var placeTextFields = []placeTextField{
	{name: "address", limit: domain.MaxPlaceAddressLength, tooLong: domain.ErrPlaceAddressTooLong,
		set: func(p *domain.Place, v *string) { p.Address = v },
		get: func(p *domain.Place) *string { return p.Address }},
	{name: "operating_hours", limit: domain.MaxPlaceOperatingHoursLength, tooLong: domain.ErrPlaceOperatingHoursTooLong,
		set: func(p *domain.Place, v *string) { p.OperatingHours = v },
		get: func(p *domain.Place) *string { return p.OperatingHours }},
	{name: "estimated_price_range", limit: domain.MaxPlaceEstimatedPriceRangeLength, tooLong: domain.ErrPlaceEstimatedPriceRangeTooLong,
		set: func(p *domain.Place, v *string) { p.EstimatedPriceRange = v },
		get: func(p *domain.Place) *string { return p.EstimatedPriceRange }},
	{name: "short_description",
		set: func(p *domain.Place, v *string) { p.ShortDescription = v },
		get: func(p *domain.Place) *string { return p.ShortDescription }},
}

func TestService_Create_OptionalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{name: "trimmed", input: "  Jl. Sudirman  ", want: new("Jl. Sudirman")},
		{name: "blank_absent", input: " \t\n ", want: nil},
	}

	for _, f := range placeTextFields {
		for _, tc := range tests {
			t.Run(f.name+"_"+tc.name, func(t *testing.T) {
				p := &domain.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
				input := tc.input
				f.set(p, &input)

				stored, err := createPlace(t, openCategory(), p)
				require.NoError(t, err, "Create(%s %q)", f.name, tc.input)
				assert.Equal(t, tc.want, f.get(stored), "Create(%s %q) stored value", f.name, tc.input)
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
			{name: "at_limit", input: strings.Repeat("a", f.limit)},
			{name: "multibyte", input: strings.Repeat("é", f.limit)},
			{name: "counted_after_trim", input: "  " + strings.Repeat("a", f.limit) + "  "},
			{name: "over_limit", input: strings.Repeat("a", f.limit+1), wantErr: f.tooLong},
		}

		for _, tc := range tests {
			t.Run(f.name+"_"+tc.name, func(t *testing.T) {
				p := &domain.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
				input := tc.input
				f.set(p, &input)

				stored, err := createPlace(t, openCategory(), p)
				assert.ErrorIs(t, err, tc.wantErr, "Create(%s of %d bytes)", f.name, len(tc.input))
				if tc.wantErr != nil {
					assert.Nil(t, stored, "Create(%s of %d bytes) must keep a refused place out of storage", f.name, len(tc.input))
				}
			})
		}
	}
}

func TestService_Update_OptionalText(t *testing.T) {
	type row struct {
		name    string
		input   string
		want    *string
		wantErr error
	}

	for _, f := range placeTextFields {
		tests := []row{
			{name: "trimmed", input: "  Jl. Sudirman  ", want: new("Jl. Sudirman")},
			{name: "blank_absent", input: " \t\n ", want: nil},
		}
		if f.limit > 0 {
			tests = append(tests, row{name: "over_limit", input: strings.Repeat("a", f.limit+1), wantErr: f.tooLong})
		}

		for _, tc := range tests {
			t.Run(f.name+"_"+tc.name, func(t *testing.T) {
				input := kopiInput()
				value := tc.input
				f.set(input, &value)

				stored, err := updatePlace(t, 7, input)
				require.ErrorIs(t, err, tc.wantErr, "Update(7, %s of %d bytes)", f.name, len(tc.input))
				if tc.wantErr != nil {
					assert.Nil(t, stored, "Update(7, %s of %d bytes) must keep a refused update out of storage", f.name, len(tc.input))
					return
				}
				assert.Equal(t, tc.want, f.get(stored), "Update(7, %s %q) stored value", f.name, tc.input)
			})
		}
	}
}

const (
	uploadKey = "uploads/places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	imageKey  = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
)

// promoted is the media Promote returns for uploadKey.
var promoted = domain.Media{Key: imageKey, ContentType: "image/webp", Size: 2048}

// fetchKopi expects one Fetch(7) on repo, which it answers with savedKopi.
func fetchKopi(repo *mock.MockRepository) *gomock.Call {
	return repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedKopi(), nil)
}

func TestService_MediaUploadURL(t *testing.T) {
	repo, media := newMocks(t)
	upload := storage.Upload{
		Key:    uploadKey,
		URL:    "http://localhost:3900/signed",
		Method: "PUT",
	}
	gomock.InOrder(
		fetchKopi(repo),
		media.EXPECT().PresignUpload(gomock.Any(), "places/7", "image/webp").Return(upload, nil),
	)

	got, err := places.NewService(repo, media).MediaUploadURL(t.Context(), 7, "image/webp")
	require.NoError(t, err, "MediaUploadURL(7, %q)", "image/webp")
	assert.Equal(t, upload, got, "MediaUploadURL(7, %q)", "image/webp")
}

func TestService_MediaUploadURL_LookupFailure(t *testing.T) {
	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown_place", getErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "lookup_failure", getErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be presigned for a place that cannot be read.
			repo, media := newMocks(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := places.NewService(repo, media).MediaUploadURL(t.Context(), 7, "image/webp")
			assert.ErrorIs(t, err, tc.wantErr, "MediaUploadURL(7)")
		})
	}
}

func TestService_MediaUploadURL_PresignFailure(t *testing.T) {
	tests := []struct {
		name       string
		presignErr error
	}{
		{name: "invalid_type", presignErr: storage.ErrInvalidMediaType},
		{name: "storage_failure", presignErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			fetchKopi(repo)
			media.EXPECT().PresignUpload(gomock.Any(), "places/7", "image/gif").Return(storage.Upload{}, tc.presignErr)

			_, err := places.NewService(repo, media).MediaUploadURL(t.Context(), 7, "image/gif")
			assert.ErrorIs(t, err, tc.presignErr, "MediaUploadURL(7, %q)", "image/gif")
		})
	}
}

// promoteOK expects one Promote(prefix "places/7", uploadKey) on media, which it answers with promoted.
func promoteOK(media *mock.MockMediaStorage) *gomock.Call {
	return media.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(promoted, nil)
}

func TestService_AddMedia(t *testing.T) {
	// The promoted media is stored as it is, and nothing is deleted.
	repo, media := newMocks(t)
	updated := savedKopi()
	updated.Medias = []domain.Media{{ID: 3, Key: imageKey, ContentType: "image/webp", Size: 2048, Position: 1}}

	gomock.InOrder(
		fetchKopi(repo),
		promoteOK(media),
		repo.EXPECT().AddMedia(gomock.Any(), int64(7), promoted).Return(updated, nil),
	)

	got, err := places.NewService(repo, media).AddMedia(t.Context(), 7, uploadKey)
	require.NoError(t, err, "AddMedia(7, %q)", uploadKey)
	assert.Same(t, updated, got, "AddMedia(7, %q) must return the stored place", uploadKey)
}

func TestService_AddMedia_LookupFailure(t *testing.T) {
	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown_place", getErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "lookup_failure", getErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be promoted for a place that cannot be read.
			repo, media := newMocks(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := places.NewService(repo, media).AddMedia(t.Context(), 7, uploadKey)
			assert.ErrorIs(t, err, tc.wantErr, "AddMedia(7, %q)", uploadKey)
		})
	}
}

func TestService_AddMedia_PromoteFailure(t *testing.T) {
	tests := []struct {
		name       string
		promoteErr error
	}{
		{name: "invalid_key", promoteErr: storage.ErrInvalidUploadKey},
		{name: "not_uploaded", promoteErr: storage.ErrMediaNotUploaded},
		{name: "invalid_type", promoteErr: storage.ErrInvalidMediaType},
		{name: "image_too_large", promoteErr: storage.ErrImageTooLarge},
		{name: "storage_failure", promoteErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be stored.
			repo, media := newMocks(t)
			fetchKopi(repo)
			media.EXPECT().Promote(gomock.Any(), "places/7", uploadKey).Return(domain.Media{}, tc.promoteErr)

			_, err := places.NewService(repo, media).AddMedia(t.Context(), 7, uploadKey)
			assert.ErrorIs(t, err, tc.promoteErr, "AddMedia(7, %q)", uploadKey)
		})
	}
}

func TestService_AddMedia_StoreFailure(t *testing.T) {
	tests := []struct {
		name      string
		addErr    error
		deleteErr error
	}{
		{name: "place_deleted", addErr: domain.ErrPlaceNotFound},
		{name: "media_limit", addErr: domain.ErrMediaLimitReached},
		{name: "storage_failure", addErr: errBoom},
		{name: "delete_failure_ignored", addErr: errBoom, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The promoted image is removed.
			repo, media := newMocks(t)
			gomock.InOrder(
				fetchKopi(repo),
				promoteOK(media),
				repo.EXPECT().AddMedia(gomock.Any(), int64(7), promoted).Return(nil, tc.addErr),
				media.EXPECT().Delete(gomock.Any(), imageKey).Return(tc.deleteErr),
			)

			_, err := places.NewService(repo, media).AddMedia(t.Context(), 7, uploadKey)
			assert.ErrorIs(t, err, tc.addErr, "AddMedia(7, %q)", uploadKey)
		})
	}
}

func TestService_DeleteMedia(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "deleted", repoErr: nil},
		{name: "unknown_media", repoErr: domain.ErrMediaNotFound},
		{name: "storage_failure", repoErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues the image for deletion, so the service
			// must not delete it itself.
			repo, media := newMocks(t)
			repo.EXPECT().DeleteMedia(gomock.Any(), int64(7), int64(3)).Return(tc.repoErr)

			err := places.NewService(repo, media).DeleteMedia(t.Context(), 7, 3)
			assert.ErrorIs(t, err, tc.repoErr, "DeleteMedia(7, 3)")
		})
	}
}

func TestService_Fetch(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "found", repoErr: nil, wantErr: nil},
		{name: "unknown_place", repoErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "storage_failure", repoErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedKopi(), tc.repoErr)

			got, err := places.NewService(repo, media).Fetch(t.Context(), 7)
			assert.ErrorIs(t, err, tc.wantErr, "Fetch(7)")
			if tc.wantErr == nil {
				require.NotNil(t, got, "Fetch(7) must return the stored place")
				assert.Equal(t, int64(7), got.ID, "Fetch(7).ID")
			}
		})
	}
}

func TestService_AddRating(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "added", repoErr: nil, wantErr: nil},
		{name: "unknown_place", repoErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "storage_failure", repoErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().AddRating(gomock.Any(), int64(7), 4).Return(tc.repoErr)

			err := places.NewService(repo, media).AddRating(t.Context(), 7, 4)
			assert.ErrorIs(t, err, tc.wantErr, "AddRating(7, 4)")
		})
	}
}

func TestService_RemoveRating(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "removed", repoErr: nil, wantErr: nil},
		{name: "unknown_place", repoErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "storage_failure", repoErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().RemoveRating(gomock.Any(), int64(7), 4).Return(tc.repoErr)

			err := places.NewService(repo, media).RemoveRating(t.Context(), 7, 4)
			assert.ErrorIs(t, err, tc.wantErr, "RemoveRating(7, 4)")
		})
	}
}

func TestService_FetchCategory(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "found", repoErr: nil, wantErr: nil},
		{name: "unknown_category", repoErr: domain.ErrCategoryNotFound, wantErr: domain.ErrCategoryNotFound},
		{name: "storage_failure", repoErr: errBoom, wantErr: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, media := newMocks(t)
			repo.EXPECT().FetchCategory(gomock.Any(), int64(1)).Return(warungCategory(), tc.repoErr)

			got, err := places.NewService(repo, media).FetchCategory(t.Context(), 1)
			assert.ErrorIs(t, err, tc.wantErr, "FetchCategory(1)")
			if tc.wantErr == nil {
				require.NotNil(t, got, "FetchCategory(1) must return the stored category")
				assert.Equal(t, int64(1), got.ID, "FetchCategory(1).ID")
				assert.Len(t, got.Fields, 6, "FetchCategory(1).Fields (Warung has 6 fields)")
			}
		})
	}
}
