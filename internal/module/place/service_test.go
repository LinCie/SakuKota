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
		{name: "name surrounding spaces are removed", catName: "  Kopi Tubruk  ", catSlug: validSlug, wantErr: nil},
		{name: "name of 100 characters", catName: strings.Repeat("é", 100), catSlug: "kopi", wantErr: nil},
		{name: "name of 101 characters", catName: strings.Repeat("é", 101), catSlug: "kopi", wantErr: place.ErrInvalidCategoryName},
		{name: "name of 20 family emoji is 100 code points", catName: strings.Repeat(family, 20), catSlug: "kopi", wantErr: nil},
		{name: "name of 21 family emoji is 105 code points", catName: strings.Repeat(family, 21), catSlug: "kopi", wantErr: place.ErrInvalidCategoryName},
		{name: "blank name", catName: "   ", catSlug: validSlug, wantErr: place.ErrInvalidCategoryName},
		{name: "empty name", catName: "", catSlug: validSlug, wantErr: place.ErrInvalidCategoryName},

		{name: "single letter slug", catName: validName, catSlug: "a", wantErr: nil},
		{name: "digits-only slug", catName: validName, catSlug: "123", wantErr: nil},
		{name: "words joined by single hyphens", catName: validName, catSlug: "kopi-tubruk-2", wantErr: nil},
		{name: "slug of 100 characters", catName: validName, catSlug: strings.Repeat("a", 100), wantErr: nil},
		{name: "slug surrounding spaces are removed", catName: validName, catSlug: "  kopi-tubruk  ", wantErr: nil},
		{name: "slug of 101 characters", catName: validName, catSlug: strings.Repeat("a", 101), wantErr: place.ErrInvalidSlug},
		{name: "uppercase slug", catName: validName, catSlug: "Kopi", wantErr: place.ErrInvalidSlug},
		{name: "underscore in slug", catName: validName, catSlug: "kopi_tubruk", wantErr: place.ErrInvalidSlug},
		{name: "leading hyphen", catName: validName, catSlug: "-kopi", wantErr: place.ErrInvalidSlug},
		{name: "trailing hyphen", catName: validName, catSlug: "kopi-", wantErr: place.ErrInvalidSlug},
		{name: "double hyphen", catName: validName, catSlug: "kopi--tubruk", wantErr: place.ErrInvalidSlug},
		{name: "space inside slug", catName: validName, catSlug: "kopi tubruk", wantErr: place.ErrInvalidSlug},
		{name: "slash in slug", catName: validName, catSlug: "kopi/teh", wantErr: place.ErrInvalidSlug},
		{name: "non-ASCII letter in slug", catName: validName, catSlug: "kopi-é", wantErr: place.ErrInvalidSlug},
		{name: "missing slug", catName: validName, catSlug: "", wantErr: place.ErrInvalidSlug},
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
		{name: "lowercase key", in: field("halal", "Halal", place.FieldTypeBoolean), wantErr: nil},
		{name: "key with a digit after the first letter", in: field("a1", "Halal", place.FieldTypeBoolean), wantErr: nil},
		{name: "key with underscores", in: field("wifi_available", "Wi-Fi", place.FieldTypeBoolean), wantErr: nil},
		{name: "key of 64 characters", in: field("a"+strings.Repeat("b", 63), "Label", place.FieldTypeText), wantErr: nil},
		{name: "key surrounding spaces are removed", in: field(" halal ", "Halal", place.FieldTypeBoolean), wantErr: nil},
		{name: "empty key", in: field("", "Halal", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldKey},
		{name: "key starts with a digit", in: field("1halal", "Halal", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldKey},
		{name: "key starts with an underscore", in: field("_halal", "Halal", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldKey},
		{name: "uppercase key", in: field("Halal", "Halal", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldKey},
		{name: "hyphen in key", in: field("wifi-available", "Wi-Fi", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldKey},
		{name: "space inside key", in: field("has space", "Halal", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldKey},
		{name: "key of 65 characters", in: field("a"+strings.Repeat("b", 64), "Label", place.FieldTypeText), wantErr: place.ErrInvalidFieldKey},

		{name: "label surrounding spaces are removed", in: field("halal", "  Halal  ", place.FieldTypeBoolean), wantErr: nil},
		{name: "label of 100 characters", in: field("halal", strings.Repeat("é", 100), place.FieldTypeBoolean), wantErr: nil},
		{name: "label of 101 characters", in: field("halal", strings.Repeat("é", 101), place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldLabel},
		{name: "label of 21 family emoji", in: field("halal", strings.Repeat(family, 21), place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldLabel},
		{name: "blank label", in: field("halal", "   ", place.FieldTypeBoolean), wantErr: place.ErrInvalidFieldLabel},

		{name: "text type", in: field("notes", "Notes", place.FieldTypeText), wantErr: nil},
		{name: "number type", in: field("seats", "Seats", place.FieldTypeNumber), wantErr: nil},
		{name: "boolean type", in: field("halal", "Halal", place.FieldTypeBoolean), wantErr: nil},
		{name: "range type", in: field("price_range", "Price", place.FieldTypeRange), wantErr: nil},
		{name: "date type", in: field("opened_on", "Opened", place.FieldTypeDate), wantErr: nil},
		{name: "unknown type", in: field("halal", "Halal", "email"), wantErr: place.ErrInvalidFieldDataType},
		{name: "type name in uppercase", in: field("halal", "Halal", "Text"), wantErr: place.ErrInvalidFieldDataType},
		{name: "empty type", in: field("halal", "Halal", ""), wantErr: place.ErrInvalidFieldDataType},

		{name: "options on a text field", in: field("notes", "Notes", place.FieldTypeText, "a"), wantErr: place.ErrFieldOptionsNotAllowed},
		{name: "options on a boolean field", in: field("halal", "Halal", place.FieldTypeBoolean, "yes"), wantErr: place.ErrFieldOptionsNotAllowed},
		{name: "select without options", in: field("payment", "Payment", place.FieldTypeSelect), wantErr: place.ErrFieldOptionsRequired},
		{name: "select with a blank option", in: field("payment", "Payment", place.FieldTypeSelect, "cash", ""), wantErr: place.ErrFieldOptionsRequired},
		{name: "select with a whitespace-only option", in: field("payment", "Payment", place.FieldTypeSelect, "cash", "   "), wantErr: place.ErrFieldOptionsRequired},
		{name: "select with a repeated option", in: field("payment", "Payment", place.FieldTypeSelect, "cash", "cash"), wantErr: place.ErrFieldOptionsRequired},
		{name: "select with an option repeated after trimming", in: field("payment", "Payment", place.FieldTypeSelect, "cash", " cash "), wantErr: place.ErrFieldOptionsRequired},
		{name: "select with options", in: field("payment", "Payment", place.FieldTypeSelect, "cash", "card"), wantErr: nil},
		{name: "select options surrounding spaces are removed", in: field("payment", "Payment", place.FieldTypeSelect, "  cash ", "card"), wantErr: nil},
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
		{name: "unknown place", getErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
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
		{name: "surrounding spaces are removed", input: "  Kopi Baru  ", wantName: "Kopi Baru", wantErr: nil},
		{name: "empty", input: "", wantName: "", wantErr: place.ErrInvalidName},
		{name: "spaces only", input: "   ", wantName: "", wantErr: place.ErrInvalidName},
		{name: "150 characters is allowed", input: long, wantName: long, wantErr: nil},
		{name: "151 characters is too long", input: long + "a", wantName: "", wantErr: place.ErrPlaceNameTooLong},
		{name: "150 multibyte characters is allowed", input: strings.Repeat("é", 150), wantName: strings.Repeat("é", 150), wantErr: nil},
		{name: "151 multibyte characters is too long", input: strings.Repeat("é", 151), wantName: "", wantErr: place.ErrPlaceNameTooLong},
		{name: "length is counted after trimming", input: "  " + long + "  ", wantName: long, wantErr: nil},
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
		{name: "deletes the place", repoErr: nil, wantErr: nil},
		{name: "unknown place", repoErr: place.ErrPlaceNotFound, wantErr: place.ErrPlaceNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
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
