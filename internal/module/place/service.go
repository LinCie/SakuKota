package place

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"rantau-go/internal/postgis"
	"rantau-go/internal/storage"
	"rantau-go/internal/text"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error)
	Create(ctx context.Context, p *Place) (*Place, error)
	GetPlace(ctx context.Context, id int64) (*Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, id int64) (*Category, error)
	// Update replaces the place's editable fields with those of p.
	Update(ctx context.Context, id int64, p *Place) (*Place, error)
	Delete(ctx context.Context, id int64) error
	// ImageUploadURL returns an upload for a new image of the place.
	ImageUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error)
	// SetImage makes the uploaded image the place's image, replacing any
	// previous one.
	SetImage(ctx context.Context, id int64, uploadKey string) (*Place, error)
}

type service struct {
	repo   Repository
	images ImageStorage
}

func NewService(repo Repository, images ImageStorage) Service {
	return &service{
		repo:   repo,
		images: images,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

func (s *service) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error) {
	return s.repo.GetMany(ctx, filter)
}

func (s *service) GetPlace(ctx context.Context, id int64) (*Place, error) {
	return s.repo.GetPlace(ctx, id)
}

func (s *service) GetCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *service) GetCategory(ctx context.Context, id int64) (*Category, error) {
	return s.repo.GetCategory(ctx, id)
}

func (s *service) Create(ctx context.Context, p *Place) (*Place, error) {
	if err := s.validate(ctx, p); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, p)
}

// Update replaces the stored place's fields with those of p. The repository
// reports ErrPlaceNotFound when there is no such place.
func (s *service) Update(ctx context.Context, id int64, p *Place) (*Place, error) {
	if err := s.validate(ctx, p); err != nil {
		return nil, err
	}
	p.ID = id

	return s.repo.Update(ctx, p)
}

// validate normalizes p and checks it against its category's fields.
func (s *service) validate(ctx context.Context, p *Place) error {
	name, err := normalizePlaceName(p.Name)
	if err != nil {
		return err
	}
	p.Name = name

	if err := normalizeOptionalFields(p); err != nil {
		return err
	}

	if err := validateLocation(p.Location); err != nil {
		return err
	}

	category, err := s.repo.GetCategory(ctx, p.CategoryID)
	if err != nil {
		return err
	}

	attributes, err := validateAttributes(category.Fields, p.Attributes)
	if err != nil {
		return err
	}
	p.Attributes = attributes

	return nil
}

// Delete removes the place. The repository queues its image for deletion.
func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// ImageUploadURL returns an upload for a new image of the place.
func (s *service) ImageUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error) {
	if _, err := s.repo.GetPlace(ctx, id); err != nil {
		return storage.Upload{}, err
	}

	return s.images.PresignUpload(ctx, imagePrefix(id), contentType)
}

// SetImage promotes the upload and stores it as the place's image. The
// repository queues the previous image for deletion. If the promoted image
// cannot be stored, its delete is attempted once and a failure is ignored: it
// only leaves an unused object behind.
func (s *service) SetImage(ctx context.Context, id int64, uploadKey string) (*Place, error) {
	if _, err := s.repo.GetPlace(ctx, id); err != nil {
		return nil, err
	}

	key, err := s.images.Promote(ctx, imagePrefix(id), uploadKey)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.SetImageKey(ctx, id, key)
	if err != nil {
		_ = s.images.Delete(ctx, key)
		return nil, err
	}

	return updated, nil
}

// imagePrefix is the storage prefix of the images of place id.
func imagePrefix(id int64) string {
	return "places/" + strconv.FormatInt(id, 10)
}

// normalizePlaceName trims name and checks it against the length of PLACES.name.
func normalizePlaceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrInvalidName
	case utf8.RuneCountInString(name) > MaxNameLength:
		return "", ErrNameTooLong
	}

	return name, nil
}

// normalizeOptionalFields trims the optional text fields of p, stores blank
// ones as absent, and checks them against the lengths of their columns.
func normalizeOptionalFields(p *Place) error {
	var ok bool

	if p.Address, ok = text.TrimOptionalMax(p.Address, MaxAddressLength); !ok {
		return ErrAddressTooLong
	}

	if p.OperatingHours, ok = text.TrimOptionalMax(p.OperatingHours, MaxOperatingHoursLength); !ok {
		return ErrOperatingHoursTooLong
	}

	if p.EstimatedPriceRange, ok = text.TrimOptionalMax(p.EstimatedPriceRange, MaxEstimatedPriceRangeLength); !ok {
		return ErrEstimatedPriceRangeTooLong
	}

	p.ShortDescription = text.TrimOptional(p.ShortDescription)

	return nil
}

// validateLocation reports ErrInvalidLocation when loc lies outside Indonesia.
// A nil location is allowed.
func validateLocation(loc *postgis.Point) error {
	if loc != nil && !postgis.InIndonesia(loc) {
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
// missing or null raw value is treated as an empty object. A blank optional
// text value is dropped like a null one.
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
		if !ok || value == nil || isBlankOptionalText(f, value) {
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

// isBlankOptionalText reports whether value is empty or whitespace-only text for
// an optional text field. Such a value is treated as absent.
func isBlankOptionalText(f CategoryField, value any) bool {
	text, ok := value.(string)
	return f.DataType == FieldTypeText && !f.Required && ok && strings.TrimSpace(text) == ""
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
