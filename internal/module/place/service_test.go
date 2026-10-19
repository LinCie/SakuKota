package place_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/place"
	"rantau-go/internal/module/place/mock"
	"rantau-go/internal/postgis"
)

// family is one emoji made of five code points: man, ZWJ, woman, ZWJ, girl.
const family = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

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

// field builds a category field input. Options are the variadic values.
func field(key, label, dataType string, options ...string) place.CategoryField {
	return place.CategoryField{Key: key, Label: label, DataType: dataType, Options: options}
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

	_, err := place.NewService(repo).Create(context.Background(), p)
	return stored, err
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

// -----------------------------------------------------------------------------
// Creating places
// -----------------------------------------------------------------------------

func TestService_CreatePlace_Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  bool
	}{
		{"surrounding spaces are removed", "  Warung Bu Ani  ", "Warung Bu Ani", false},
		{"tabs and newlines are removed", "\tWarung Bu Ani\n", "Warung Bu Ani", false},
		{"inner spacing is kept", "Warung  Bu Ani", "Warung  Bu Ani", false},
		{"empty", "", "", true},
		{"spaces only", "   ", "", true},
		{"tabs and newlines only", "\t\n", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, openCategory(), &place.Place{
				Name: tc.input, SubDistrictID: 1, CategoryID: 2,
			})
			if tc.wantErr {
				if !errors.Is(err, place.ErrInvalidName) {
					t.Fatalf("err = %v, want ErrInvalidName", err)
				}
				if stored != nil {
					t.Fatalf("blank name reached storage as %q", stored.Name)
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

func TestService_CreatePlace_Attributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string // stored JSON when the attributes are accepted
		wantErr    bool
	}{
		{"required field only", `{"contact":"0812-3456"}`, `{"contact":"0812-3456"}`, false},
		{"every type with a valid value",
			`{"contact":"0812","seats":12,"halal":true,"payment":"card","price_range":{"min":10000,"max":50000},"opened_on":"2024-01-31"}`,
			`{"contact":"0812","seats":12,"halal":true,"payment":"card","price_range":{"min":10000,"max":50000},"opened_on":"2024-01-31"}`, false},
		{"range with equal bounds", `{"contact":"x","price_range":{"min":5,"max":5}}`, `{"contact":"x","price_range":{"min":5,"max":5}}`, false},
		{"integer keeps every digit", `{"contact":"x","seats":12345678901234567890}`, `{"contact":"x","seats":12345678901234567890}`, false},
		{"decimal keeps its trailing zero", `{"contact":"x","seats":1.50}`, `{"contact":"x","seats":1.50}`, false},
		{"text is kept verbatim",
			`{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`,
			`{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`, false},
		{"optional null is dropped", `{"contact":"x","seats":null,"halal":null}`, `{"contact":"x"}`, false},

		{"required field missing", `{"seats":1}`, "", true},
		{"required text is only spaces", `{"contact":"   "}`, "", true},
		{"required field is null", `{"contact":null}`, "", true},
		{"null attributes miss the required field", `null`, "", true},
		{"empty object misses the required field", `{}`, "", true},
		{"array instead of object", `[1,2]`, "", true},
		{"string instead of object", `"text"`, "", true},
		{"malformed JSON", `{"contact":`, "", true},
		{"unknown key", `{"contact":"x","parking":true}`, "", true},
		{"text given a number", `{"contact":5}`, "", true},
		{"number given a string", `{"contact":"x","seats":"12"}`, "", true},
		{"number given a boolean", `{"contact":"x","seats":true}`, "", true},
		{"boolean given a string", `{"contact":"x","halal":"true"}`, "", true},
		{"boolean given a number", `{"contact":"x","halal":1}`, "", true},
		{"select value outside its options", `{"contact":"x","payment":"bitcoin"}`, "", true},
		{"select given a number", `{"contact":"x","payment":1}`, "", true},
		{"range with min above max", `{"contact":"x","price_range":{"min":9,"max":1}}`, "", true},
		{"range missing max", `{"contact":"x","price_range":{"min":1}}`, "", true},
		{"range with an extra key", `{"contact":"x","price_range":{"min":1,"max":2,"step":1}}`, "", true},
		{"range with string bounds", `{"contact":"x","price_range":{"min":"1","max":"2"}}`, "", true},
		{"range given as an array", `{"contact":"x","price_range":[1,2]}`, "", true},
		{"date with slashes", `{"contact":"x","opened_on":"2024/01/31"}`, "", true},
		{"date that does not exist", `{"contact":"x","opened_on":"2024-02-30"}`, "", true},
		{"date given as a number", `{"contact":"x","opened_on":20240131}`, "", true},
		{"date with a time", `{"contact":"x","opened_on":"2024-01-31T00:00:00Z"}`, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, warungCategory(), &place.Place{
				Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1,
				Attributes: json.RawMessage(tc.attributes),
			})
			if tc.wantErr {
				if !errors.Is(err, place.ErrInvalidAttributes) {
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

func TestService_CreatePlace_AttributesWithoutRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantStored string
	}{
		{"absent attributes are stored as an empty object", "", `{}`},
		{"null attributes are stored as an empty object", `null`, `{}`},
		{"empty object is kept", `{}`, `{}`},
		{"set optional field is kept", `{"seats":5}`, `{"seats":5}`},
		{"null optional field is dropped", `{"halal":null}`, `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createPlace(t, openCategory(), &place.Place{
				Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2,
				Attributes: json.RawMessage(tc.attributes),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSON(t, stored.Attributes, tc.wantStored)
		})
	}
}

func TestService_CreatePlace_Location(t *testing.T) {
	tests := []struct {
		name     string
		lon, lat float64
		omit     bool // no location in the request
		wantErr  bool
	}{
		{"Jakarta", 106.8, -6.2, false, false},
		{"Sabang, the western tip", 95.3, 5.9, false, false},
		{"Merauke, the eastern tip", 140.4, -8.5, false, false},
		{"west edge of the box", 95, -6.2, false, false},
		{"east edge of the box", 141, -6.2, false, false},
		{"north edge of the box", 106.8, 6, false, false},
		{"south edge of the box", 106.8, -11, false, false},
		{"location omitted", 0, 0, true, false},
		{"Tokyo", 139.7, 35.7, false, true},
		{"London", -0.1, 51.5, false, true},
		{"null island", 0, 0, false, true},
		{"just west of the box", 94.99, -6.2, false, true},
		{"just east of the box", 141.01, -6.2, false, true},
		{"just north of the box", 106.8, 6.01, false, true},
		{"just south of the box", 106.8, -11.01, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &place.Place{Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2}
			if !tc.omit {
				point, err := postgis.NewPoint(tc.lon, tc.lat)
				if err != nil {
					t.Fatalf("invalid test point: %v", err)
				}
				p.Location = point
			}

			stored, err := createPlace(t, openCategory(), p)
			if tc.wantErr {
				if err == nil {
					t.Fatal("location outside Indonesia was accepted")
				}
				if stored != nil {
					t.Fatal("location outside Indonesia reached storage")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.omit {
				if stored.Location != nil {
					t.Errorf("omitted location stored as %v", stored.Location)
				}
				return
			}
			if stored.Location == nil || stored.Location.X() != tc.lon || stored.Location.Y() != tc.lat {
				t.Errorf("stored location = %v, want [%v, %v]", stored.Location, tc.lon, tc.lat)
			}
		})
	}
}

func TestService_CreatePlace_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		lookupErr error // returned when the category is looked up
		storeErr  error // returned when the place is stored
		wantErr   error // nil means the call succeeds
	}{
		{"unknown category stops before storage", place.ErrCategoryNotFound, nil, place.ErrCategoryNotFound},
		{"storage reports an unknown sub-district", nil, place.ErrSubDistrictNotFound, place.ErrSubDistrictNotFound},
		{"storage reports an unknown category", nil, place.ErrCategoryNotFound, place.ErrCategoryNotFound},
		{"storage failure is returned", nil, boom, boom},
		{"stored place is returned", nil, nil, nil},
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

			got, err := place.NewService(repo).Create(context.Background(), &place.Place{
				Name: "Kopi Senja", SubDistrictID: 1, CategoryID: 2,
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && got.ID != 7 {
				t.Errorf("returned ID = %d, want the stored place's ID 7", got.ID)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Creating categories
// -----------------------------------------------------------------------------

func TestService_CreateCategory(t *testing.T) {
	const validName = "Kopi Tubruk"
	const validSlug = "kopi-tubruk"

	tests := []struct {
		name    string
		catName string
		catSlug string
		wantErr error
	}{
		{"name surrounding spaces are removed", "  Kopi Tubruk  ", validSlug, nil},
		{"name of 100 characters", strings.Repeat("é", 100), "kopi", nil},
		{"name of 101 characters", strings.Repeat("é", 101), "kopi", place.ErrInvalidCategoryName},
		{"name of 20 family emoji is 100 code points", strings.Repeat(family, 20), "kopi", nil},
		{"name of 21 family emoji is 105 code points", strings.Repeat(family, 21), "kopi", place.ErrInvalidCategoryName},
		{"blank name", "   ", validSlug, place.ErrInvalidCategoryName},
		{"empty name", "", validSlug, place.ErrInvalidCategoryName},

		{"single letter slug", validName, "a", nil},
		{"digits-only slug", validName, "123", nil},
		{"words joined by single hyphens", validName, "kopi-tubruk-2", nil},
		{"slug of 100 characters", validName, strings.Repeat("a", 100), nil},
		{"slug surrounding spaces are removed", validName, "  kopi-tubruk  ", nil},
		{"slug of 101 characters", validName, strings.Repeat("a", 101), place.ErrInvalidSlug},
		{"uppercase slug", validName, "Kopi", place.ErrInvalidSlug},
		{"underscore in slug", validName, "kopi_tubruk", place.ErrInvalidSlug},
		{"leading hyphen", validName, "-kopi", place.ErrInvalidSlug},
		{"trailing hyphen", validName, "kopi-", place.ErrInvalidSlug},
		{"double hyphen", validName, "kopi--tubruk", place.ErrInvalidSlug},
		{"space inside slug", validName, "kopi tubruk", place.ErrInvalidSlug},
		{"slash in slug", validName, "kopi/teh", place.ErrInvalidSlug},
		{"non-ASCII letter in slug", validName, "kopi-é", place.ErrInvalidSlug},
		{"missing slug", validName, "", place.ErrInvalidSlug},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			var stored *place.Category
			repo.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, c *place.Category) (*place.Category, error) {
					stored = c
					return c, nil
				}).AnyTimes()

			_, err := place.NewService(repo).CreateCategory(context.Background(), &place.Category{
				Name: tc.catName, Slug: tc.catSlug,
			})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatal("rejected category reached storage")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored == nil {
				t.Fatal("accepted category was not stored")
			}
			if stored.Name != strings.TrimSpace(tc.catName) {
				t.Errorf("stored name = %q, want %q", stored.Name, strings.TrimSpace(tc.catName))
			}
			if stored.Slug != strings.TrimSpace(tc.catSlug) {
				t.Errorf("stored slug = %q, want %q", stored.Slug, strings.TrimSpace(tc.catSlug))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Adding fields to categories
//
// SortOrder is not set here on purpose. Its rules are checked over HTTP and
// against the database, so changing its type does not affect these tests.
// -----------------------------------------------------------------------------

func TestService_CreateCategoryField(t *testing.T) {
	tests := []struct {
		name    string
		in      place.CategoryField
		wantErr error
	}{
		{"lowercase key", field("halal", "Halal", place.FieldTypeBoolean), nil},
		{"key with a digit after the first letter", field("a1", "Halal", place.FieldTypeBoolean), nil},
		{"key with underscores", field("wifi_available", "Wi-Fi", place.FieldTypeBoolean), nil},
		{"key of 64 characters", field("a"+strings.Repeat("b", 63), "Label", place.FieldTypeText), nil},
		{"key surrounding spaces are removed", field(" halal ", "Halal", place.FieldTypeBoolean), nil},
		{"empty key", field("", "Halal", place.FieldTypeBoolean), place.ErrInvalidFieldKey},
		{"key starts with a digit", field("1halal", "Halal", place.FieldTypeBoolean), place.ErrInvalidFieldKey},
		{"key starts with an underscore", field("_halal", "Halal", place.FieldTypeBoolean), place.ErrInvalidFieldKey},
		{"uppercase key", field("Halal", "Halal", place.FieldTypeBoolean), place.ErrInvalidFieldKey},
		{"hyphen in key", field("wifi-available", "Wi-Fi", place.FieldTypeBoolean), place.ErrInvalidFieldKey},
		{"space inside key", field("has space", "Halal", place.FieldTypeBoolean), place.ErrInvalidFieldKey},
		{"key of 65 characters", field("a"+strings.Repeat("b", 64), "Label", place.FieldTypeText), place.ErrInvalidFieldKey},

		{"label surrounding spaces are removed", field("halal", "  Halal  ", place.FieldTypeBoolean), nil},
		{"label of 100 characters", field("halal", strings.Repeat("é", 100), place.FieldTypeBoolean), nil},
		{"label of 101 characters", field("halal", strings.Repeat("é", 101), place.FieldTypeBoolean), place.ErrInvalidFieldLabel},
		{"label of 21 family emoji", field("halal", strings.Repeat(family, 21), place.FieldTypeBoolean), place.ErrInvalidFieldLabel},
		{"blank label", field("halal", "   ", place.FieldTypeBoolean), place.ErrInvalidFieldLabel},

		{"text type", field("notes", "Notes", place.FieldTypeText), nil},
		{"number type", field("seats", "Seats", place.FieldTypeNumber), nil},
		{"boolean type", field("halal", "Halal", place.FieldTypeBoolean), nil},
		{"range type", field("price_range", "Price", place.FieldTypeRange), nil},
		{"date type", field("opened_on", "Opened", place.FieldTypeDate), nil},
		{"unknown type", field("halal", "Halal", "email"), place.ErrInvalidFieldDataType},
		{"type name in uppercase", field("halal", "Halal", "Text"), place.ErrInvalidFieldDataType},
		{"empty type", field("halal", "Halal", ""), place.ErrInvalidFieldDataType},

		{"options on a text field", field("notes", "Notes", place.FieldTypeText, "a"), place.ErrFieldOptionsNotAllowed},
		{"options on a boolean field", field("halal", "Halal", place.FieldTypeBoolean, "yes"), place.ErrFieldOptionsNotAllowed},
		{"select without options", field("payment", "Payment", place.FieldTypeSelect), place.ErrFieldOptionsRequired},
		{"select with a blank option", field("payment", "Payment", place.FieldTypeSelect, "cash", ""), place.ErrFieldOptionsRequired},
		{"select with a whitespace-only option", field("payment", "Payment", place.FieldTypeSelect, "cash", "   "), place.ErrFieldOptionsRequired},
		{"select with a repeated option", field("payment", "Payment", place.FieldTypeSelect, "cash", "cash"), place.ErrFieldOptionsRequired},
		{"select with an option repeated after trimming", field("payment", "Payment", place.FieldTypeSelect, "cash", " cash "), place.ErrFieldOptionsRequired},
		{"select with options", field("payment", "Payment", place.FieldTypeSelect, "cash", "card"), nil},
		{"select options surrounding spaces are removed", field("payment", "Payment", place.FieldTypeSelect, "  cash ", "card"), nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Expected values are computed before the call, because the
			// service normalizes the input in place.
			wantKey := strings.TrimSpace(tc.in.Key)
			wantLabel := strings.TrimSpace(tc.in.Label)
			var wantOptions []string
			for _, option := range tc.in.Options {
				wantOptions = append(wantOptions, strings.TrimSpace(option))
			}

			repo := mock.NewMockRepository(gomock.NewController(t))
			var stored *place.CategoryField
			repo.EXPECT().CreateCategoryField(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, f *place.CategoryField) (*place.CategoryField, error) {
					stored = f
					return f, nil
				}).AnyTimes()

			in := tc.in
			_, err := place.NewService(repo).CreateCategoryField(context.Background(), &in)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatal("rejected field reached storage")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored == nil {
				t.Fatal("accepted field was not stored")
			}
			if stored.Key != wantKey {
				t.Errorf("stored key = %q, want %q", stored.Key, wantKey)
			}
			if stored.Label != wantLabel {
				t.Errorf("stored label = %q, want %q", stored.Label, wantLabel)
			}
			if len(wantOptions) > 0 && !slices.Equal([]string(stored.Options), wantOptions) {
				t.Errorf("stored options = %v, want %v", stored.Options, wantOptions)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Updating and deleting places
// -----------------------------------------------------------------------------

// present is a PATCH field the client sent with the value v.
func present[T any](v T) place.Optional[T] {
	return place.Optional[T]{Set: true, Value: &v}
}

// null is a PATCH field the client sent as null.
func null[T any]() place.Optional[T] {
	return place.Optional[T]{Set: true}
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
		Category:            place.Category{ID: 2, Name: "Kopi", Slug: "kopi"},
	}
}

// lookupCategory answers category lookups for the update tests. Category 1 is
// Warung, which requires "contact". Category 2 is Kopi.
func lookupCategory(_ context.Context, id int64) (*place.Category, error) {
	switch id {
	case 1:
		return warungCategory(), nil
	case 2:
		return openCategory(), nil
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

	_, err := place.NewService(repo).Update(context.Background(), id, patch)
	return stored, err
}

func TestService_UpdatePlace_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{"unknown place", place.ErrPlaceNotFound, place.ErrPlaceNotFound},
		{"lookup failure is returned", boom, boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No Update expectation: nothing may be stored when the place cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := place.NewService(repo).Update(context.Background(), 7,
				place.PlacePatch{Name: present("Kopi Baru")})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestService_UpdatePlace_Fields(t *testing.T) {
	tests := []struct {
		name   string
		patch  place.PlacePatch
		change func(*place.Place) // turns savedKopi into the wanted stored place
	}{
		{"empty patch changes nothing", place.PlacePatch{}, func(*place.Place) {}},
		{"name is trimmed", place.PlacePatch{Name: present("  Kopi Baru\t")},
			func(p *place.Place) { p.Name = "Kopi Baru" }},
		{"address is replaced", place.PlacePatch{Address: present("Jl. Kemang 2")},
			func(p *place.Place) { p.Address = ptr("Jl. Kemang 2") }},
		{"location is replaced", place.PlacePatch{Location: present(*postgis.MustPoint(106.85, -6.18))},
			func(p *place.Place) { p.Location = postgis.MustPoint(106.85, -6.18) }},
		{"sub-district is replaced", place.PlacePatch{SubDistrictID: present(int64(2))},
			func(p *place.Place) { p.SubDistrictID = 2 }},
		{"null clears every nullable field", place.PlacePatch{
			Address:             null[string](),
			Location:            null[postgis.Point](),
			OperatingHours:      null[string](),
			EstimatedPriceRange: null[string](),
			ShortDescription:    null[string](),
		}, func(p *place.Place) {
			p.Address, p.Location, p.OperatingHours = nil, nil, nil
			p.EstimatedPriceRange, p.ShortDescription = nil, nil
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := savedKopi()
			tc.change(want)

			stored, err := updatePlace(t, savedKopi(), 7, tc.patch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(stored, want) {
				t.Errorf("stored = %+v, want %+v", stored, want)
			}
		})
	}
}

func TestService_UpdatePlace_Name(t *testing.T) {
	long := strings.Repeat("a", 150)

	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  error
	}{
		{"surrounding spaces are removed", "  Kopi Baru  ", "Kopi Baru", nil},
		{"empty", "", "", place.ErrInvalidName},
		{"spaces only", "   ", "", place.ErrInvalidName},
		{"150 characters is allowed", long, long, nil},
		{"151 characters is too long", long + "a", "", place.ErrPlaceNameTooLong},
		{"150 multibyte characters is allowed", strings.Repeat("é", 150), strings.Repeat("é", 150), nil},
		{"151 multibyte characters is too long", strings.Repeat("é", 151), "", place.ErrPlaceNameTooLong},
		{"length is counted after trimming", "  " + long + "  ", long, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updatePlace(t, savedKopi(), 7, place.PlacePatch{Name: present(tc.input)})
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

func TestService_UpdatePlace_Location(t *testing.T) {
	tests := []struct {
		name    string
		lon     float64
		lat     float64
		wantErr error
	}{
		{"inside Indonesia", 106.8, -6.2, nil},
		{"on the western edge", 95, -2, nil},
		{"west of the Indonesia box", 94.9, -2, place.ErrInvalidLocation},
		{"north of the Indonesia box", 110, 6.5, place.ErrInvalidLocation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updatePlace(t, savedKopi(), 7,
				place.PlacePatch{Location: present(*postgis.MustPoint(tc.lon, tc.lat))})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatalf("location outside Indonesia reached storage")
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

func TestService_UpdatePlace_Attributes(t *testing.T) {
	// The place is a Kopi, which has no required fields.
	tests := []struct {
		name      string
		attrs     place.Optional[json.RawMessage]
		wantAttrs string
		wantErr   error
	}{
		{"new attributes replace the old ones", present(json.RawMessage(`{"halal":false}`)), `{"halal":false}`, nil},
		{"null attributes become an empty object", null[json.RawMessage](), `{}`, nil},
		{"unknown key is rejected", present(json.RawMessage(`{"parking":true}`)), "", place.ErrInvalidAttributes},
		{"wrong type is rejected", present(json.RawMessage(`{"seats":"many"}`)), "", place.ErrInvalidAttributes},
		{"array is rejected", present(json.RawMessage(`[1]`)), "", place.ErrInvalidAttributes},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updatePlace(t, savedKopi(), 7, place.PlacePatch{Attributes: tc.attrs})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatalf("invalid attributes reached storage as %s", stored.Attributes)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSON(t, stored.Attributes, tc.wantAttrs)
		})
	}
}

func TestService_UpdatePlace_Category(t *testing.T) {
	tests := []struct {
		name           string
		attrs          string // the place's attributes before the update
		patch          place.PlacePatch
		wantCategoryID int64
		wantAttrs      string
		wantErr        error
	}{
		{"attributes that fit the new category are kept", `{"contact":"0812","seats":12}`,
			place.PlacePatch{CategoryID: present(int64(1))}, 1, `{"contact":"0812","seats":12}`, nil},
		{"attributes missing a required field of the new category are rejected", `{"halal":true,"seats":12}`,
			place.PlacePatch{CategoryID: present(int64(1))}, 0, "", place.ErrInvalidAttributes},
		{"attributes unknown to the new category are rejected", `{"contact":"0812"}`,
			place.PlacePatch{CategoryID: present(int64(2))}, 0, "", place.ErrInvalidAttributes},
		{"new attributes are checked against the new category", `{"contact":"0812"}`,
			place.PlacePatch{CategoryID: present(int64(1)), Attributes: present(json.RawMessage(`{"seats":12}`))},
			0, "", place.ErrInvalidAttributes},
		{"new attributes that fit the new category are stored", `{"halal":true}`,
			place.PlacePatch{CategoryID: present(int64(1)), Attributes: present(json.RawMessage(`{"contact":"0812"}`))},
			1, `{"contact":"0812"}`, nil},
		{"unknown category is rejected", `{"halal":true}`,
			place.PlacePatch{CategoryID: present(int64(99))}, 0, "", place.ErrCategoryNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing := savedKopi()
			existing.Attributes = json.RawMessage(tc.attrs)

			stored, err := updatePlace(t, existing, 7, tc.patch)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatalf("rejected update reached storage")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored.CategoryID != tc.wantCategoryID {
				t.Errorf("stored category = %d, want %d", stored.CategoryID, tc.wantCategoryID)
			}
			assertJSON(t, stored.Attributes, tc.wantAttrs)
		})
	}
}

func TestService_UpdatePlace_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		updateErr error // returned when the place is stored
		wantErr   error // nil means the call succeeds
	}{
		{"storage reports the place is gone", place.ErrPlaceNotFound, place.ErrPlaceNotFound},
		{"storage reports an unknown sub-district", place.ErrSubDistrictNotFound, place.ErrSubDistrictNotFound},
		{"storage failure is returned", boom, boom},
		{"stored place is returned", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetPlace(gomock.Any(), int64(7)).Return(savedKopi(), nil)
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&place.Place{ID: 7}, tc.updateErr)

			got, err := place.NewService(repo).Update(context.Background(), 7,
				place.PlacePatch{Name: present("Kopi Baru")})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && got.ID != 7 {
				t.Errorf("returned ID = %d, want the stored place's ID 7", got.ID)
			}
		})
	}
}

func TestService_DeletePlace(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{"deletes the place", nil, nil},
		{"unknown place", place.ErrPlaceNotFound, place.ErrPlaceNotFound},
		{"storage failure is returned", boom, boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := place.NewService(repo).Delete(context.Background(), 7)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
