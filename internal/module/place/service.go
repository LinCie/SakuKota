package place

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"rantau-go/internal/postgis"
)

var (
	ErrInvalidName            = errors.New("name is required")
	ErrPlaceNameTooLong       = errors.New("name must be 150 characters or fewer")
	ErrPlaceNotFound          = errors.New("place not found")
	ErrInvalidCategoryName    = errors.New("name must be 1-100 characters")
	ErrInvalidSlug            = errors.New("slug must be 1-100 lowercase letters or digits, with single hyphens between words")
	ErrSubDistrictNotFound    = errors.New("sub-district not found")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrCategoryExists         = errors.New("category with this name or slug already exists")
	ErrInvalidFieldKey        = errors.New("key must be 1-64 lowercase letters, digits, or underscores, starting with a letter")
	ErrInvalidFieldLabel      = errors.New("label must be 1-100 characters")
	ErrInvalidFieldDataType   = errors.New("data_type must be one of text, number, boolean, select, range, date")
	ErrFieldOptionsRequired   = errors.New("select fields require unique, non-blank options")
	ErrFieldOptionsNotAllowed = errors.New("options are only allowed for select fields")
	ErrInvalidFieldSortOrder  = errors.New("sort_order must be a positive position, or omitted to add the field last")
	ErrCategoryFieldExists    = errors.New("field with this key already exists in the category")
	ErrInvalidAttributes      = errors.New("invalid attributes")
	ErrInvalidLocation        = errors.New("location must be within Indonesia")
	ErrIdempotencyKeyReused   = errors.New("idempotency key was already used for a different request")
)

var (
	fieldKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	slugPattern     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// Indonesia's bounding box in degrees: west and east longitude, south and north latitude.
const (
	indonesiaMinLongitude = 95
	indonesiaMaxLongitude = 141
	indonesiaMinLatitude  = -11
	indonesiaMaxLatitude  = 6
)

// maxNameLength is the size of the VARCHAR(100) columns holding category names
// and slugs and field labels. VARCHAR counts characters, not bytes.
const maxNameLength = 100

// maxPlaceNameLength is the size of the VARCHAR(150) column holding place names.
const maxPlaceNameLength = 150

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error)
	Create(ctx context.Context, p *Place) (*Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	CreateCategory(ctx context.Context, c *Category) (*Category, error)
	CreateCategoryField(ctx context.Context, f *CategoryField) (*CategoryField, error)
	Update(ctx context.Context, id int64, patch PlacePatch) (*Place, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

func (s *service) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, error) {
	return s.repo.GetMany(ctx, filter)
}

func (s *service) GetCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *service) Create(ctx context.Context, p *Place) (*Place, error) {
	name, err := normalizePlaceName(p.Name)
	if err != nil {
		return nil, err
	}
	p.Name = name

	if err := validateLocation(p.Location); err != nil {
		return nil, err
	}

	category, err := s.repo.GetCategory(ctx, p.CategoryID)
	if err != nil {
		return nil, err
	}

	attributes, err := validateAttributes(category.Fields, p.Attributes)
	if err != nil {
		return nil, err
	}
	p.Attributes = attributes

	return s.repo.Create(ctx, p)
}

func (s *service) CreateCategory(ctx context.Context, c *Category) (*Category, error) {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" || utf8.RuneCountInString(c.Name) > maxNameLength {
		return nil, ErrInvalidCategoryName
	}

	c.Slug = strings.TrimSpace(c.Slug)
	if utf8.RuneCountInString(c.Slug) > maxNameLength || !slugPattern.MatchString(c.Slug) {
		return nil, ErrInvalidSlug
	}

	return s.repo.CreateCategory(ctx, c)
}

func (s *service) CreateCategoryField(ctx context.Context, f *CategoryField) (*CategoryField, error) {
	f.Key = strings.TrimSpace(f.Key)
	if !fieldKeyPattern.MatchString(f.Key) {
		return nil, ErrInvalidFieldKey
	}

	f.Label = strings.TrimSpace(f.Label)
	if f.Label == "" || utf8.RuneCountInString(f.Label) > maxNameLength {
		return nil, ErrInvalidFieldLabel
	}

	if f.SortOrder != nil && *f.SortOrder < 1 {
		return nil, ErrInvalidFieldSortOrder
	}

	switch f.DataType {
	case FieldTypeText, FieldTypeNumber, FieldTypeBoolean, FieldTypeRange, FieldTypeDate:
		if len(f.Options) > 0 {
			return nil, ErrFieldOptionsNotAllowed
		}
		f.Options = nil
	case FieldTypeSelect:
		if len(f.Options) == 0 {
			return nil, ErrFieldOptionsRequired
		}
		for i, option := range f.Options {
			option = strings.TrimSpace(option)
			if option == "" || slices.Contains(f.Options[:i], option) {
				return nil, ErrFieldOptionsRequired
			}
			f.Options[i] = option
		}
	default:
		return nil, ErrInvalidFieldDataType
	}

	return s.repo.CreateCategoryField(ctx, f)
}

// Update applies the patch to the stored place. Attributes are checked against
// the place's category whenever either of them changes.
func (s *service) Update(ctx context.Context, id int64, patch PlacePatch) (*Place, error) {
	p, err := s.repo.GetPlace(ctx, id)
	if err != nil {
		return nil, err
	}

	if patch.Name.Set && patch.Name.Value != nil {
		name, err := normalizePlaceName(*patch.Name.Value)
		if err != nil {
			return nil, err
		}
		p.Name = name
	}

	if patch.SubDistrictID.Set && patch.SubDistrictID.Value != nil {
		p.SubDistrictID = *patch.SubDistrictID.Value
	}

	if patch.CategoryID.Set && patch.CategoryID.Value != nil {
		p.CategoryID = *patch.CategoryID.Value
	}

	if patch.Address.Set {
		p.Address = patch.Address.Value
	}

	if patch.Location.Set {
		if err := validateLocation(patch.Location.Value); err != nil {
			return nil, err
		}
		p.Location = patch.Location.Value
	}

	if patch.OperatingHours.Set {
		p.OperatingHours = patch.OperatingHours.Value
	}

	if patch.EstimatedPriceRange.Set {
		p.EstimatedPriceRange = patch.EstimatedPriceRange.Value
	}

	if patch.ShortDescription.Set {
		p.ShortDescription = patch.ShortDescription.Value
	}

	if patch.CategoryID.Set || patch.Attributes.Set {
		category, err := s.repo.GetCategory(ctx, p.CategoryID)
		if err != nil {
			return nil, err
		}

		raw := p.Attributes
		if patch.Attributes.Set {
			raw = nil
			if patch.Attributes.Value != nil {
				raw = *patch.Attributes.Value
			}
		}

		attributes, err := validateAttributes(category.Fields, raw)
		if err != nil {
			return nil, err
		}
		p.Attributes = attributes
	}

	return s.repo.Update(ctx, p)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// normalizePlaceName trims name and checks it against the length of PLACES.name.
func normalizePlaceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrInvalidName
	case utf8.RuneCountInString(name) > maxPlaceNameLength:
		return "", ErrPlaceNameTooLong
	}

	return name, nil
}

// validateLocation reports ErrInvalidLocation when loc lies outside Indonesia.
// A nil location is allowed.
func validateLocation(loc *postgis.Point) error {
	if loc != nil && (loc.X() < indonesiaMinLongitude || loc.X() > indonesiaMaxLongitude ||
		loc.Y() < indonesiaMinLatitude || loc.Y() > indonesiaMaxLatitude) {
		return ErrInvalidLocation
	}

	return nil
}

// -----------------------------------------------------------------------------
// Attribute validation
// -----------------------------------------------------------------------------

// validateAttributes checks raw attribute values against the category's
// fields and returns them re-encoded as a JSON object. Required fields must be
// present, and keys that are not fields of the category are rejected. A
// missing or null raw value is treated as an empty object.
func validateAttributes(fields []CategoryField, raw json.RawMessage) (json.RawMessage, error) {
	var values map[string]any
	if len(raw) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		// Keep numbers as json.Number so they are stored exactly as sent.
		decoder.UseNumber()
		if err := decoder.Decode(&values); err != nil {
			return nil, fmt.Errorf("%w: must be a JSON object", ErrInvalidAttributes)
		}
	}
	if values == nil {
		values = make(map[string]any)
	}

	for _, f := range fields {
		value, ok := values[f.Key]
		if !ok || value == nil {
			if f.Required {
				return nil, fmt.Errorf("%w: %q is required", ErrInvalidAttributes, f.Key)
			}
			delete(values, f.Key)
			continue
		}

		if err := validateAttribute(f, value); err != nil {
			return nil, err
		}
	}

	for key := range values {
		known := slices.ContainsFunc(fields, func(f CategoryField) bool {
			return f.Key == key
		})
		if !known {
			return nil, fmt.Errorf("%w: unknown field %q", ErrInvalidAttributes, key)
		}
	}

	return json.Marshal(values)
}

// validateAttribute checks a single value against its field's data type.
func validateAttribute(f CategoryField, value any) error {
	switch f.DataType {
	case FieldTypeText:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: %q must be text", ErrInvalidAttributes, f.Key)
		}
		if f.Required && strings.TrimSpace(text) == "" {
			return fmt.Errorf("%w: %q is required", ErrInvalidAttributes, f.Key)
		}
	case FieldTypeNumber:
		if _, ok := value.(json.Number); !ok {
			return fmt.Errorf("%w: %q must be a number", ErrInvalidAttributes, f.Key)
		}
	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: %q must be a boolean", ErrInvalidAttributes, f.Key)
		}
	case FieldTypeSelect:
		option, ok := value.(string)
		if !ok || !slices.Contains(f.Options, option) {
			return fmt.Errorf("%w: %q must be one of: %s",
				ErrInvalidAttributes, f.Key, strings.Join(f.Options, ", "))
		}
	case FieldTypeRange:
		if !validRange(value) {
			return fmt.Errorf(`%w: %q must be {"min": number, "max": number} with min <= max`,
				ErrInvalidAttributes, f.Key)
		}
	case FieldTypeDate:
		date, ok := value.(string)
		if _, err := time.Parse(time.DateOnly, date); !ok || err != nil {
			return fmt.Errorf("%w: %q must be a date in YYYY-MM-DD format", ErrInvalidAttributes, f.Key)
		}
	}

	return nil
}

// validRange reports whether value is an object with exactly the numeric keys
// "min" and "max", where min <= max.
func validRange(value any) bool {
	bounds, ok := value.(map[string]any)
	if !ok || len(bounds) != 2 {
		return false
	}

	lowNumber, ok := bounds["min"].(json.Number)
	if !ok {
		return false
	}

	highNumber, ok := bounds["max"].(json.Number)
	if !ok {
		return false
	}

	low, err := lowNumber.Float64()
	if err != nil {
		return false
	}

	high, err := highNumber.Float64()
	if err != nil {
		return false
	}

	return low <= high
}
