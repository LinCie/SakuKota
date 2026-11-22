package subdistricts_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/domain"
	"rantau-go/internal/module/subdistricts"
	"rantau-go/internal/module/subdistricts/mock"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// family is one emoji made of five code points: man, ZWJ, woman, ZWJ, girl.
const family = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

// newService returns a Service over mocks of its repository and media storage.
// Any call without an expectation fails the test.
func newService(t *testing.T) (*subdistricts.Service, *mock.MockRepository, *mock.MockMediaStorage) {
	t.Helper()

	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	media := mock.NewMockMediaStorage(ctrl)
	return subdistricts.NewService(repo, media), repo, media
}

// createSubDistrict runs Service.Create for sd. It returns the sub-district the
// repository was asked to store, or nil if storage was never reached.
func createSubDistrict(t *testing.T, sd *domain.SubDistrict) (*domain.SubDistrict, error) {
	t.Helper()

	svc, repo, _ := newService(t)
	var stored *domain.SubDistrict
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.SubDistrict) (*domain.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := svc.Create(t.Context(), sd)
	return stored, err
}

func TestService_Create_Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  error // nil means the name is accepted
	}{
		{name: "trim-spaces", input: "  Menteng  ", wantName: "Menteng"},
		{name: "trim-tabs-newlines", input: "\tMenteng\n", wantName: "Menteng"},
		{name: "keep-inner-spacing", input: "Kebayoran  Baru", wantName: "Kebayoran  Baru"},
		{name: "empty", input: "", wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "spaces-only", input: "   ", wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "100-chars", input: strings.Repeat("a", 100), wantName: strings.Repeat("a", 100)},
		{name: "101-chars", input: strings.Repeat("a", 101), wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "100-multibyte", input: strings.Repeat("é", 100), wantName: strings.Repeat("é", 100)},
		{name: "101-multibyte", input: strings.Repeat("é", 101), wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "20-family-emoji", input: strings.Repeat(family, 20), wantName: strings.Repeat(family, 20)},
		{name: "21-family-emoji", input: strings.Repeat(family, 21), wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "length-after-trim", input: "  " + strings.Repeat("a", 100) + "  ", wantName: strings.Repeat("a", 100)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createSubDistrict(t, &domain.SubDistrict{Name: tc.input, Slug: "menteng"})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Create(%s)", tc.name)
				assert.Nil(t, stored, "Create(%s) reached storage with an invalid name", tc.name)
				return
			}
			require.NoError(t, err, "Create(%s)", tc.name)
			require.NotNil(t, stored, "Create(%s) did not store the accepted sub-district", tc.name)
			assert.Equal(t, tc.wantName, stored.Name, "Create(%s) stored name", tc.name)
		})
	}
}

func TestService_Create_Slug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSlug string
		wantErr  error // nil means the slug is accepted
	}{
		{name: "trim-spaces", input: "  menteng  ", wantSlug: "menteng"},
		{name: "100-chars", input: strings.Repeat("a", 100), wantSlug: strings.Repeat("a", 100)},
		{name: "101-chars", input: strings.Repeat("a", 101), wantErr: domain.ErrSubDistrictSlugInvalid},
		{name: "empty", input: "", wantErr: domain.ErrSubDistrictSlugInvalid},
		{name: "spaces-only", input: "   ", wantErr: domain.ErrSubDistrictSlugInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createSubDistrict(t, &domain.SubDistrict{Name: "Menteng", Slug: tc.input})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Create(%s)", tc.name)
				assert.Nil(t, stored, "Create(%s) reached storage with an invalid slug", tc.name)
				return
			}
			require.NoError(t, err, "Create(%s)", tc.name)
			require.NotNil(t, stored, "Create(%s) did not store the accepted sub-district", tc.name)
			assert.Equal(t, tc.wantSlug, stored.Slug, "Create(%s) stored slug", tc.name)
		})
	}
}

func TestService_Create_OptionalFieldLimits(t *testing.T) {
	tests := []struct {
		name    string
		in      domain.SubDistrict
		wantErr error // nil means the sub-district is accepted
	}{
		{name: "spending-zero", in: domain.SubDistrict{EstimatedDailySpending: new(0.0)}},
		{name: "spending-cents", in: domain.SubDistrict{EstimatedDailySpending: new(1500000.5)}},
		{name: "spending-largest", in: domain.SubDistrict{EstimatedDailySpending: new(99999999.99)}},
		{name: "spending-negative", in: domain.SubDistrict{EstimatedDailySpending: new(-0.01)}, wantErr: domain.ErrSubDistrictSpendingInvalid},
		{name: "spending-over-decimal", in: domain.SubDistrict{EstimatedDailySpending: new(100000000.0)}, wantErr: domain.ErrSubDistrictSpendingInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			in.Name, in.Slug = "Menteng", "menteng"

			stored, err := createSubDistrict(t, &in)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Create(%s)", tc.name)
				assert.Nil(t, stored, "Create(%s) reached storage with a rejected sub-district", tc.name)
				return
			}
			require.NoError(t, err, "Create(%s)", tc.name)
			assert.NotNil(t, stored, "Create(%s) did not store the accepted sub-district", tc.name)
		})
	}
}

func TestService_OptionalText(t *testing.T) {
	fields := []struct {
		name string
		set  func(*domain.SubDistrict, *string)
		get  func(*domain.SubDistrict) *string
	}{
		{name: "flood-history",
			set: func(sd *domain.SubDistrict, v *string) { sd.FloodHistory = v },
			get: func(sd *domain.SubDistrict) *string { return sd.FloodHistory }},
		{name: "safety",
			set: func(sd *domain.SubDistrict, v *string) { sd.SafetyCrimeInformation = v },
			get: func(sd *domain.SubDistrict) *string { return sd.SafetyCrimeInformation }},
	}

	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{name: "trim", input: "  Banjir 2020\n", want: new("Banjir 2020")},
		{name: "keep-inner-spacing", input: "Banjir  2020", want: new("Banjir  2020")},
		{name: "blank-is-absent", input: " \t\n ", want: nil},
	}

	for _, f := range fields {
		for _, tc := range tests {
			t.Run("create-"+f.name+"-"+tc.name, func(t *testing.T) {
				sd := &domain.SubDistrict{Name: "Menteng", Slug: "menteng"}
				input := tc.input
				f.set(sd, &input)

				stored, err := createSubDistrict(t, sd)
				require.NoError(t, err, "Create(%s %q)", f.name, tc.input)
				require.NotNil(t, stored, "Create(%s %q) did not store the sub-district", f.name, tc.input)
				assert.Equal(t, tc.want, f.get(stored), "Create(%s %q) stored text", f.name, tc.input)
			})

			t.Run("update-"+f.name+"-"+tc.name, func(t *testing.T) {
				sd := mentengInput()
				input := tc.input
				f.set(sd, &input)

				stored, err := updateSubDistrict(t, 7, sd)
				require.NoError(t, err, "Update(%s %q)", f.name, tc.input)
				require.NotNil(t, stored, "Update(%s %q) did not store the sub-district", f.name, tc.input)
				assert.Equal(t, tc.want, f.get(stored), "Update(%s %q) stored text", f.name, tc.input)
			})
		}
	}
}

func TestService_Create_StoresFieldsAsSent(t *testing.T) {
	tests := []struct {
		name string
		in   domain.SubDistrict
	}{
		{name: "every-field", in: domain.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           new("Banjir tahunan"),
			SafetyCrimeInformation: new("Aman di siang hari"),
			EstimatedDailySpending: new(1500000.5),
		}},
		{name: "name-and-slug", in: domain.SubDistrict{Name: "Menteng", Slug: "menteng"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			stored, err := createSubDistrict(t, &in)
			require.NoError(t, err, "Create(%s)", tc.name)
			require.NotNil(t, stored, "Create(%s) did not store the accepted sub-district", tc.name)
			assert.Equal(t, tc.in, *stored, "Create(%s) stored sub-district", tc.name)
		})
	}
}

func TestService_Create_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name     string
		storeErr error // returned when the sub-district is stored
		wantErr  error // nil means the call succeeds
	}{
		{name: "duplicate", storeErr: domain.ErrSubDistrictExists, wantErr: domain.ErrSubDistrictExists},
		{name: "storage-failure", storeErr: boom, wantErr: boom},
		{name: "stored", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			repo.EXPECT().Create(gomock.Any(), &domain.SubDistrict{Name: "Menteng", Slug: "menteng"}).
				Return(&domain.SubDistrict{ID: 7}, tc.storeErr)

			got, err := svc.Create(t.Context(), &domain.SubDistrict{Name: "Menteng", Slug: "menteng"})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Create(%s)", tc.name)
				return
			}
			require.NoError(t, err, "Create(%s)", tc.name)
			require.NotNil(t, got, "Create(%s) = nil, want the stored sub-district", tc.name)
			assert.EqualValues(t, 7, got.ID, "Create(%s).ID", tc.name)
		})
	}
}

// savedMenteng is the stored sub-district that updates start from: a Menteng
// with every optional field and a media set.
func savedMenteng() *domain.SubDistrict {
	return &domain.SubDistrict{
		ID:                     7,
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           new("Banjir 2020"),
		SafetyCrimeInformation: new("Aman di siang hari"),
		EstimatedDailySpending: new(1500000.5),
		Medias: []domain.Media{
			{ID: 3, Key: "subdistricts/7/medias/9f8e7d6c-5b4a-4321-8fed-cba987654321.png", ContentType: "image/png", Size: 1024, Position: 1},
		},
	}
}

// mentengInput is a full replacement for savedMenteng: a Menteng with every
// optional field set. Updates change one field of it at a time.
func mentengInput() *domain.SubDistrict {
	return &domain.SubDistrict{
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           new("Banjir 2020"),
		SafetyCrimeInformation: new("Aman di siang hari"),
		EstimatedDailySpending: new(1500000.5),
	}
}

// updateSubDistrict runs Service.Update for sub-district id with the
// replacement sd. It returns the sub-district the repository was asked to
// store, or nil if storage was never reached.
func updateSubDistrict(t *testing.T, id int64, sd *domain.SubDistrict) (*domain.SubDistrict, error) {
	t.Helper()

	svc, repo, _ := newService(t)
	var stored *domain.SubDistrict
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.SubDistrict) (*domain.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := svc.Update(t.Context(), id, sd)
	return stored, err
}

func TestService_Fetch(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "found", repoErr: nil, wantErr: nil},
		{name: "unknown", repoErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "storage-failure", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var found *domain.SubDistrict
			if tc.repoErr == nil {
				found = savedMenteng()
			}

			svc, repo, _ := newService(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(found, tc.repoErr)

			got, err := svc.Fetch(t.Context(), 7)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Fetch(7)")
				return
			}
			require.NoError(t, err, "Fetch(7)")
			require.NotNil(t, got, "Fetch(7) = nil, want the sub-district")
			assert.EqualValues(t, 7, got.ID, "Fetch(7).ID")
		})
	}
}

func TestService_Update_Fields(t *testing.T) {
	tests := []struct {
		name   string
		change func(*domain.SubDistrict) // turns mentengInput into the replacement
		want   func(*domain.SubDistrict) // turns mentengInput into the wanted stored sub-district
	}{
		{name: "same-fields", change: func(*domain.SubDistrict) {}, want: func(*domain.SubDistrict) {}},
		{name: "name-trimmed",
			change: func(sd *domain.SubDistrict) { sd.Name = "  Menteng Atas\t" },
			want:   func(sd *domain.SubDistrict) { sd.Name = "Menteng Atas" }},
		{name: "name-100-chars",
			change: func(sd *domain.SubDistrict) { sd.Name = strings.Repeat("a", 100) },
			want:   func(sd *domain.SubDistrict) { sd.Name = strings.Repeat("a", 100) }},
		{name: "slug-trimmed",
			change: func(sd *domain.SubDistrict) { sd.Slug = "  menteng-atas " },
			want:   func(sd *domain.SubDistrict) { sd.Slug = "menteng-atas" }},
		{name: "flood-history-replaced",
			change: func(sd *domain.SubDistrict) { sd.FloodHistory = new("Banjir 2024") },
			want:   func(sd *domain.SubDistrict) { sd.FloodHistory = new("Banjir 2024") }},
		{name: "safety-replaced",
			change: func(sd *domain.SubDistrict) { sd.SafetyCrimeInformation = new("Rawan malam") },
			want:   func(sd *domain.SubDistrict) { sd.SafetyCrimeInformation = new("Rawan malam") }},
		{name: "spending-replaced",
			change: func(sd *domain.SubDistrict) { sd.EstimatedDailySpending = new(750000.25) },
			want:   func(sd *domain.SubDistrict) { sd.EstimatedDailySpending = new(750000.25) }},
		{name: "spending-zero",
			change: func(sd *domain.SubDistrict) { sd.EstimatedDailySpending = new(0.0) },
			want:   func(sd *domain.SubDistrict) { sd.EstimatedDailySpending = new(0.0) }},
		{name: "nullable-cleared",
			change: func(sd *domain.SubDistrict) {
				sd.FloodHistory, sd.SafetyCrimeInformation, sd.EstimatedDailySpending = nil, nil, nil
			},
			want: func(sd *domain.SubDistrict) {
				sd.FloodHistory, sd.SafetyCrimeInformation, sd.EstimatedDailySpending = nil, nil, nil
			}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := mentengInput()
			want.ID = 7
			tc.want(want)

			input := mentengInput()
			tc.change(input)

			stored, err := updateSubDistrict(t, 7, input)
			require.NoError(t, err, "Update(7, %s)", tc.name)
			assert.Equal(t, want, stored, "Update(7, %s) stored sub-district", tc.name)
		})
	}
}

func TestService_Update_ID(t *testing.T) {
	input := mentengInput()
	input.ID = 99 // the sub-district comes from the path, not from the replacement

	stored, err := updateSubDistrict(t, 7, input)
	require.NoError(t, err, "Update(7)")
	require.NotNil(t, stored, "Update(7) did not store the sub-district")
	assert.EqualValues(t, 7, stored.ID, "Update(7) stored ID")
}

func TestService_Update_Rejected(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*domain.SubDistrict)
		wantErr error
	}{
		{name: "name-spaces-only", change: func(sd *domain.SubDistrict) { sd.Name = "   " }, wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "name-101-chars", change: func(sd *domain.SubDistrict) { sd.Name = strings.Repeat("a", 101) }, wantErr: domain.ErrSubDistrictNameInvalid},
		{name: "slug-empty", change: func(sd *domain.SubDistrict) { sd.Slug = "" }, wantErr: domain.ErrSubDistrictSlugInvalid},
		{name: "slug-101-chars", change: func(sd *domain.SubDistrict) { sd.Slug = strings.Repeat("a", 101) }, wantErr: domain.ErrSubDistrictSlugInvalid},
		{name: "spending-negative", change: func(sd *domain.SubDistrict) { sd.EstimatedDailySpending = new(-0.01) },
			wantErr: domain.ErrSubDistrictSpendingInvalid},
		{name: "spending-over-decimal", change: func(sd *domain.SubDistrict) { sd.EstimatedDailySpending = new(100000000.0) },
			wantErr: domain.ErrSubDistrictSpendingInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := mentengInput()
			tc.change(input)

			stored, err := updateSubDistrict(t, 7, input)
			assert.ErrorIs(t, err, tc.wantErr, "Update(7, %s)", tc.name)
			assert.Nil(t, stored, "Update(7, %s) reached storage with a rejected update", tc.name)
		})
	}
}

func TestService_Update_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		updateErr error // returned when the sub-district is stored
		wantErr   error // nil means the call succeeds
	}{
		{name: "gone", updateErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "duplicate", updateErr: domain.ErrSubDistrictExists, wantErr: domain.ErrSubDistrictExists},
		{name: "storage-failure", updateErr: boom, wantErr: boom},
		{name: "stored", updateErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _ := newService(t)
			stored := mentengInput()
			stored.ID = 7
			repo.EXPECT().Update(gomock.Any(), stored).Return(&domain.SubDistrict{ID: 7}, tc.updateErr)

			got, err := svc.Update(t.Context(), 7, mentengInput())
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Update(7, %s)", tc.name)
				return
			}
			require.NoError(t, err, "Update(7, %s)", tc.name)
			require.NotNil(t, got, "Update(7, %s) = nil, want the stored sub-district", tc.name)
			assert.EqualValues(t, 7, got.ID, "Update(7, %s).ID", tc.name)
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
		{name: "deleted", repoErr: nil, wantErr: nil},
		{name: "unknown", repoErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "in-use", repoErr: domain.ErrSubDistrictInUse, wantErr: domain.ErrSubDistrictInUse},
		{name: "storage-failure", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues the images, so the service deletes none:
			// the media storage mock has no expectations.
			svc, repo, _ := newService(t)
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := svc.Delete(t.Context(), 7)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "Delete(7)")
				return
			}
			assert.NoError(t, err, "Delete(7)")
		})
	}
}

func TestService_Delete_QueuesImages(t *testing.T) {
	// The repository queues every image for deletion, so the service must not
	// delete any of them itself: the media storage mock has no expectations.
	svc, repo, _ := newService(t)
	repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil)

	assert.NoError(t, svc.Delete(t.Context(), 7), "Delete(7)")
}

// slotCase is one of a sub-district's image slots, with its storage prefix
// and keys for sub-district 7.
type slotCase struct {
	slot      subdistricts.MediaSlot
	prefix    string
	uploadKey string
	newKey    string
}

var (
	mediasSlot = slotCase{
		slot:      subdistricts.SlotMedias,
		prefix:    "subdistricts/7/medias",
		uploadKey: "uploads/subdistricts/7/medias/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
		newKey:    "subdistricts/7/medias/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
	}
	slotCases = []slotCase{mediasSlot}
)

func TestService_MediaUploadURL(t *testing.T) {
	for _, sc := range slotCases {
		t.Run(string(sc.slot), func(t *testing.T) {
			svc, repo, media := newService(t)
			upload := storage.Upload{
				Key:    sc.uploadKey,
				URL:    "http://localhost:3900/signed",
				Method: "PUT",
			}
			gomock.InOrder(
				repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				media.EXPECT().PresignUpload(gomock.Any(), sc.prefix, "image/webp").Return(upload, nil),
			)

			got, err := svc.MediaUploadURL(t.Context(), 7, sc.slot, "image/webp")
			require.NoError(t, err, "MediaUploadURL(7, %q)", sc.slot)
			assert.Equal(t, upload, got, "MediaUploadURL(7, %q)", sc.slot)
		})
	}
}

func TestService_MediaUploadURL_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown", getErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "lookup-failure", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be presigned for a sub-district that cannot be read:
			// the media storage mock has no expectations.
			svc, repo, _ := newService(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := svc.MediaUploadURL(t.Context(), 7, subdistricts.SlotMedias, "image/webp")
			assert.ErrorIs(t, err, tc.wantErr, "MediaUploadURL(7)")
		})
	}
}

func TestService_MediaUploadURL_PresignFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		presignErr error
	}{
		{name: "invalid-type", presignErr: storage.ErrInvalidMediaType},
		{name: "storage-failure", presignErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, media := newService(t)
			gomock.InOrder(
				repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				media.EXPECT().PresignUpload(gomock.Any(), "subdistricts/7/medias", "image/gif").Return(storage.Upload{}, tc.presignErr),
			)

			_, err := svc.MediaUploadURL(t.Context(), 7, subdistricts.SlotMedias, "image/gif")
			assert.ErrorIs(t, err, tc.presignErr, "MediaUploadURL(7)")
		})
	}
}

// storeCase is an operation that promotes an upload and stores the image.
type storeCase struct {
	name string
	sc   slotCase
	// expectStore expects the repository method that stores the image to be
	// called once with sub-district 7 and the promoted media.
	expectStore func(repo *mock.MockRepository, promoted domain.Media) *gomock.Call
	run         func(s *subdistricts.Service, uploadKey string) (*domain.SubDistrict, error)
}

func storeCases(t *testing.T) []storeCase {
	return []storeCase{
		{
			name: "add-media",
			sc:   mediasSlot,
			expectStore: func(repo *mock.MockRepository, promoted domain.Media) *gomock.Call {
				return repo.EXPECT().AddMedia(gomock.Any(), int64(7), promoted)
			},
			run: func(s *subdistricts.Service, uploadKey string) (*domain.SubDistrict, error) {
				return s.AddMedia(t.Context(), 7, uploadKey)
			},
		},
	}
}

// promotedMedia is the media Promote returns for an upload stored under key.
func promotedMedia(key string) domain.Media {
	return domain.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

func TestService_StoreMedia(t *testing.T) {
	for _, op := range storeCases(t) {
		t.Run(op.name, func(t *testing.T) {
			// The repository queues any replaced image, so the service deletes
			// nothing: MediaStorage.Delete has no expectation.
			svc, repo, media := newService(t)
			promoted := promotedMedia(op.sc.newKey)
			updated := savedMenteng()

			gomock.InOrder(
				repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				media.EXPECT().Promote(gomock.Any(), op.sc.prefix, op.sc.uploadKey).Return(promoted, nil),
				op.expectStore(repo, promoted).Return(updated, nil),
			)

			got, err := op.run(svc, op.sc.uploadKey)
			require.NoError(t, err, op.name)
			assert.Same(t, updated, got, "%s should return the stored sub-district", op.name)
		})
	}
}

func TestService_StoreMedia_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown", getErr: domain.ErrSubDistrictNotFound, wantErr: domain.ErrSubDistrictNotFound},
		{name: "lookup-failure", getErr: boom, wantErr: boom},
	}

	for _, op := range storeCases(t) {
		for _, tc := range tests {
			t.Run(op.name+"-"+tc.name, func(t *testing.T) {
				// Nothing may be promoted for a sub-district that cannot be read:
				// the media storage mock has no expectations.
				svc, repo, _ := newService(t)
				repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, tc.getErr)

				_, err := op.run(svc, op.sc.uploadKey)
				assert.ErrorIs(t, err, tc.wantErr, op.name)
			})
		}
	}
}

func TestService_StoreMedia_PromoteFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		promoteErr error
	}{
		{name: "invalid-key", promoteErr: storage.ErrInvalidUploadKey},
		{name: "not-uploaded", promoteErr: storage.ErrMediaNotUploaded},
		{name: "invalid-type", promoteErr: storage.ErrInvalidMediaType},
		{name: "too-large", promoteErr: storage.ErrImageTooLarge},
		{name: "storage-failure", promoteErr: boom},
	}

	for _, op := range storeCases(t) {
		for _, tc := range tests {
			t.Run(op.name+"-"+tc.name, func(t *testing.T) {
				// Nothing may be stored: the store method has no expectation.
				svc, repo, media := newService(t)
				gomock.InOrder(
					repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
					media.EXPECT().Promote(gomock.Any(), op.sc.prefix, op.sc.uploadKey).Return(domain.Media{}, tc.promoteErr),
				)

				_, err := op.run(svc, op.sc.uploadKey)
				assert.ErrorIs(t, err, tc.promoteErr, op.name)
			})
		}
	}
}

func TestService_StoreMedia_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		storeErr  error
		deleteErr error
	}{
		{name: "deleted-meanwhile", storeErr: domain.ErrSubDistrictNotFound},
		{name: "limit-reached", storeErr: domain.ErrMediaLimitReached},
		{name: "storage-failure", storeErr: boom},
		{name: "delete-failure-ignored", storeErr: boom, deleteErr: errors.New("timeout")},
	}

	for _, op := range storeCases(t) {
		for _, tc := range tests {
			t.Run(op.name+"-"+tc.name, func(t *testing.T) {
				// The promoted image is removed.
				svc, repo, media := newService(t)
				promoted := promotedMedia(op.sc.newKey)

				gomock.InOrder(
					repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
					media.EXPECT().Promote(gomock.Any(), op.sc.prefix, op.sc.uploadKey).Return(promoted, nil),
					op.expectStore(repo, promoted).Return(nil, tc.storeErr),
					media.EXPECT().Delete(gomock.Any(), op.sc.newKey).Return(tc.deleteErr),
				)

				_, err := op.run(svc, op.sc.uploadKey)
				assert.ErrorIs(t, err, tc.storeErr, op.name)
			})
		}
	}
}

func TestService_DeleteMedia(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "deleted", repoErr: nil},
		{name: "unknown", repoErr: domain.ErrMediaNotFound},
		{name: "storage-failure", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The repository queues the image for deletion, so the service
			// must not delete it itself: the media storage mock has no
			// expectations.
			svc, repo, _ := newService(t)
			repo.EXPECT().DeleteMedia(gomock.Any(), int64(7), int64(3)).Return(tc.repoErr)

			err := svc.DeleteMedia(t.Context(), 7, 3)
			if tc.repoErr != nil {
				assert.ErrorIs(t, err, tc.repoErr, "DeleteMedia(7, 3)")
				return
			}
			assert.NoError(t, err, "DeleteMedia(7, 3)")
		})
	}
}

func TestService_SetBoundary(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "stored", repoErr: nil},
		{name: "unknown", repoErr: domain.ErrSubDistrictNotFound},
		{name: "crosses-itself", repoErr: domain.ErrSubDistrictBoundaryInvalid},
		{name: "storage-failure", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boundary := mustBoundary(t, squareJSON)
			var stored *domain.SubDistrict
			if tc.repoErr == nil {
				stored = savedMenteng()
			}

			svc, repo, _ := newService(t)
			repo.EXPECT().SetBoundary(gomock.Any(), int64(7), boundary).Return(stored, tc.repoErr)

			got, err := svc.SetBoundary(t.Context(), 7, boundary)
			if tc.repoErr != nil {
				assert.ErrorIs(t, err, tc.repoErr, "SetBoundary(7)")
			} else {
				assert.NoError(t, err, "SetBoundary(7)")
			}
			assert.Same(t, stored, got, "SetBoundary(7)")
		})
	}
}

func TestService_SetBoundary_Rejected(t *testing.T) {
	tests := []struct {
		name     string
		boundary *postgis.MultiPolygon
	}{
		{name: "none", boundary: nil},
		{name: "vertex-east", boundary: mustBoundary(t,
			`{"type":"Polygon","coordinates":[[[106.8,-6.2],[141.01,-6.2],[106.9,-6.3],[106.8,-6.2]]]}`)},
		{name: "polygon-outside", boundary: mustBoundary(t,
			`{"type":"MultiPolygon","coordinates":[[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.2]]],`+
				`[[[0,0],[1,0],[1,1],[0,0]]]]}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a rejected boundary must never reach storage.
			svc, _, _ := newService(t)

			got, err := svc.SetBoundary(t.Context(), 7, tc.boundary)
			assert.ErrorIs(t, err, domain.ErrSubDistrictBoundaryInvalid, "SetBoundary(7, %s)", tc.name)
			assert.Nil(t, got, "SetBoundary(7, %s)", tc.name)
		})
	}
}

func TestService_SetBoundary_EdgesOfIndonesia(t *testing.T) {
	// The box's edges count as inside.
	boundary := mustBoundary(t, `{"type":"Polygon","coordinates":[[[95,-11],[141,-11],[141,6],[95,6],[95,-11]]]}`)

	svc, repo, _ := newService(t)
	repo.EXPECT().SetBoundary(gomock.Any(), int64(7), boundary).Return(savedMenteng(), nil)

	_, err := svc.SetBoundary(t.Context(), 7, boundary)
	assert.NoError(t, err, "SetBoundary(7)")
}
