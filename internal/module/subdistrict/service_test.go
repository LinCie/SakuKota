package subdistrict_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/subdistrict"
	"rantau-go/internal/module/subdistrict/mock"
	"rantau-go/internal/storage"
)

// family is one emoji made of five code points: man, ZWJ, woman, ZWJ, girl.
const family = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

func ptr[T any](v T) *T {
	return &v
}

// createSubDistrict runs Service.Create for sd. It returns the sub-district the
// repository was asked to store, or nil if storage was never reached.
func createSubDistrict(t *testing.T, sd *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	var stored *subdistrict.SubDistrict
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := subdistrict.NewService(repo, noImages(t)).Create(context.Background(), sd)
	return stored, err
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
// Creating sub-districts
// -----------------------------------------------------------------------------

func TestService_Create_Name(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  error // nil means the name is accepted
	}{
		{name: "surrounding spaces are removed", input: "  Menteng  ", wantName: "Menteng"},
		{name: "tabs and newlines are removed", input: "\tMenteng\n", wantName: "Menteng"},
		{name: "inner spacing is kept", input: "Kebayoran  Baru", wantName: "Kebayoran  Baru"},
		{name: "empty", input: "", wantErr: subdistrict.ErrInvalidName},
		{name: "spaces only", input: "   ", wantErr: subdistrict.ErrInvalidName},
		{name: "100 characters", input: strings.Repeat("a", 100), wantName: strings.Repeat("a", 100)},
		{name: "101 characters", input: strings.Repeat("a", 101), wantErr: subdistrict.ErrInvalidName},
		{name: "100 multibyte characters", input: strings.Repeat("é", 100), wantName: strings.Repeat("é", 100)},
		{name: "101 multibyte characters", input: strings.Repeat("é", 101), wantErr: subdistrict.ErrInvalidName},
		{name: "20 family emoji are 100 code points", input: strings.Repeat(family, 20), wantName: strings.Repeat(family, 20)},
		{name: "21 family emoji are 105 code points", input: strings.Repeat(family, 21), wantErr: subdistrict.ErrInvalidName},
		{name: "length is counted after trimming", input: "  " + strings.Repeat("a", 100) + "  ", wantName: strings.Repeat("a", 100)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createSubDistrict(t, &subdistrict.SubDistrict{Name: tc.input, Slug: "menteng"})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "invalid name reached storage")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored, "accepted sub-district was not stored")
			assert.Equal(t, tc.wantName, stored.Name)
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
		{name: "surrounding spaces are removed", input: "  menteng  ", wantSlug: "menteng"},
		{name: "100 characters", input: strings.Repeat("a", 100), wantSlug: strings.Repeat("a", 100)},
		{name: "101 characters", input: strings.Repeat("a", 101), wantErr: subdistrict.ErrInvalidSlug},
		{name: "empty", input: "", wantErr: subdistrict.ErrInvalidSlug},
		{name: "spaces only", input: "   ", wantErr: subdistrict.ErrInvalidSlug},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createSubDistrict(t, &subdistrict.SubDistrict{Name: "Menteng", Slug: tc.input})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "invalid slug reached storage")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored, "accepted sub-district was not stored")
			assert.Equal(t, tc.wantSlug, stored.Slug)
		})
	}
}

func TestService_Create_OptionalFieldLimits(t *testing.T) {
	tests := []struct {
		name    string
		in      subdistrict.SubDistrict
		wantErr error // nil means the sub-district is accepted
	}{
		{name: "estimated daily spending of zero", in: subdistrict.SubDistrict{EstimatedDailySpending: ptr(0.0)}},
		{name: "estimated daily spending with cents", in: subdistrict.SubDistrict{EstimatedDailySpending: ptr(1500000.5)}},
		{name: "largest estimated daily spending", in: subdistrict.SubDistrict{EstimatedDailySpending: ptr(99999999.99)}},
		{name: "negative estimated daily spending", in: subdistrict.SubDistrict{EstimatedDailySpending: ptr(-0.01)},
			wantErr: subdistrict.ErrInvalidEstimatedDailySpending},
		{name: "estimated daily spending too large for DECIMAL(10,2)", in: subdistrict.SubDistrict{EstimatedDailySpending: ptr(100000000.0)},
			wantErr: subdistrict.ErrInvalidEstimatedDailySpending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			in.Name, in.Slug = "Menteng", "menteng"

			stored, err := createSubDistrict(t, &in)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, stored, "rejected sub-district reached storage")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored, "accepted sub-district was not stored")
		})
	}
}

func TestService_OptionalText(t *testing.T) {
	fields := []struct {
		name string
		set  func(*subdistrict.SubDistrict, *string)
		get  func(*subdistrict.SubDistrict) *string
	}{
		{name: "flood history",
			set: func(sd *subdistrict.SubDistrict, v *string) { sd.FloodHistory = v },
			get: func(sd *subdistrict.SubDistrict) *string { return sd.FloodHistory }},
		{name: "safety and crime information",
			set: func(sd *subdistrict.SubDistrict, v *string) { sd.SafetyCrimeInformation = v },
			get: func(sd *subdistrict.SubDistrict) *string { return sd.SafetyCrimeInformation }},
	}

	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{name: "surrounding whitespace is removed", input: "  Banjir 2020\n", want: ptr("Banjir 2020")},
		{name: "inner spacing is kept", input: "Banjir  2020", want: ptr("Banjir  2020")},
		{name: "blank is stored as absent", input: " \t\n ", want: nil},
	}

	for _, f := range fields {
		for _, tc := range tests {
			t.Run("create/"+f.name+"/"+tc.name, func(t *testing.T) {
				sd := &subdistrict.SubDistrict{Name: "Menteng", Slug: "menteng"}
				input := tc.input
				f.set(sd, &input)

				stored, err := createSubDistrict(t, sd)
				require.NoError(t, err)
				assert.Equal(t, ptrString(tc.want), ptrString(f.get(stored)), "stored %s", f.name)
			})

			t.Run("update/"+f.name+"/"+tc.name, func(t *testing.T) {
				sd := mentengInput()
				input := tc.input
				f.set(sd, &input)

				stored, err := updateSubDistrict(t, 7, sd)
				require.NoError(t, err)
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

func TestService_Create_StoresFieldsAsSent(t *testing.T) {
	tests := []struct {
		name string
		in   subdistrict.SubDistrict
	}{
		{name: "every field set", in: subdistrict.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           ptr("Banjir tahunan"),
			SafetyCrimeInformation: ptr("Aman di siang hari"),
			EstimatedDailySpending: ptr(1500000.5),
		}},
		{name: "only name and slug", in: subdistrict.SubDistrict{Name: "Menteng", Slug: "menteng"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			stored, err := createSubDistrict(t, &in)
			require.NoError(t, err)
			require.NotNil(t, stored, "accepted sub-district was not stored")
			assert.Equal(t, tc.in, *stored)
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
		{name: "duplicate name or slug is returned", storeErr: subdistrict.ErrSubDistrictExists, wantErr: subdistrict.ErrSubDistrictExists},
		{name: "storage failure is returned", storeErr: boom, wantErr: boom},
		{name: "stored sub-district is returned", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&subdistrict.SubDistrict{ID: 7}, tc.storeErr)

			got, err := subdistrict.NewService(repo, noImages(t)).Create(context.Background(), &subdistrict.SubDistrict{
				Name: "Menteng", Slug: "menteng",
			})
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored sub-district was not returned")
				assert.Equal(t, int64(7), got.ID)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Reading, updating, and deleting sub-districts
// -----------------------------------------------------------------------------

// savedMenteng is the stored sub-district that updates start from: a Menteng
// with every optional field and both images set.
func savedMenteng() *subdistrict.SubDistrict {
	return &subdistrict.SubDistrict{
		ID:                     7,
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           ptr("Banjir 2020"),
		SafetyCrimeInformation: ptr("Aman di siang hari"),
		EstimatedDailySpending: ptr(1500000.5),
		ImageKey:               ptr(oldImageKey),
		StaticMapImageKey:      ptr(oldStaticMapKey),
	}
}

// mentengInput is a full replacement for savedMenteng: a Menteng with every
// optional field set. Updates change one field of it at a time.
func mentengInput() *subdistrict.SubDistrict {
	return &subdistrict.SubDistrict{
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           ptr("Banjir 2020"),
		SafetyCrimeInformation: ptr("Aman di siang hari"),
		EstimatedDailySpending: ptr(1500000.5),
	}
}

// updateSubDistrict runs Service.Update for sub-district id with the
// replacement sd. It returns the sub-district the repository was asked to
// store, or nil if storage was never reached.
func updateSubDistrict(t *testing.T, id int64, sd *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))

	var stored *subdistrict.SubDistrict
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := subdistrict.NewService(repo, noImages(t)).Update(context.Background(), id, sd)
	return stored, err
}

func TestService_GetSubDistrict(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "returns the sub-district", repoErr: nil, wantErr: nil},
		{name: "unknown sub-district", repoErr: subdistrict.ErrSubDistrictNotFound, wantErr: subdistrict.ErrSubDistrictNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var found *subdistrict.SubDistrict
			if tc.repoErr == nil {
				found = savedMenteng()
			}

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(found, tc.repoErr)

			got, err := subdistrict.NewService(repo, noImages(t)).GetSubDistrict(context.Background(), 7)
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "sub-district was not returned")
				assert.Equal(t, int64(7), got.ID)
			}
		})
	}
}

func TestService_Update_Fields(t *testing.T) {
	tests := []struct {
		name   string
		change func(*subdistrict.SubDistrict) // turns mentengInput into the replacement
		want   func(*subdistrict.SubDistrict) // turns mentengInput into the wanted stored sub-district
	}{
		{name: "same fields change nothing", change: func(*subdistrict.SubDistrict) {}, want: func(*subdistrict.SubDistrict) {}},
		{name: "name is trimmed",
			change: func(sd *subdistrict.SubDistrict) { sd.Name = "  Menteng Atas\t" },
			want:   func(sd *subdistrict.SubDistrict) { sd.Name = "Menteng Atas" }},
		{name: "name of 100 characters is allowed",
			change: func(sd *subdistrict.SubDistrict) { sd.Name = strings.Repeat("a", 100) },
			want:   func(sd *subdistrict.SubDistrict) { sd.Name = strings.Repeat("a", 100) }},
		{name: "slug is trimmed",
			change: func(sd *subdistrict.SubDistrict) { sd.Slug = "  menteng-atas " },
			want:   func(sd *subdistrict.SubDistrict) { sd.Slug = "menteng-atas" }},
		{name: "flood history is replaced",
			change: func(sd *subdistrict.SubDistrict) { sd.FloodHistory = ptr("Banjir 2024") },
			want:   func(sd *subdistrict.SubDistrict) { sd.FloodHistory = ptr("Banjir 2024") }},
		{name: "safety information is replaced",
			change: func(sd *subdistrict.SubDistrict) { sd.SafetyCrimeInformation = ptr("Rawan malam") },
			want:   func(sd *subdistrict.SubDistrict) { sd.SafetyCrimeInformation = ptr("Rawan malam") }},
		{name: "estimated daily spending is replaced",
			change: func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(750000.25) },
			want:   func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(750000.25) }},
		{name: "estimated daily spending of zero is allowed",
			change: func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(0.0) },
			want:   func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(0.0) }},
		{name: "absent nullable fields are cleared",
			change: func(sd *subdistrict.SubDistrict) {
				sd.FloodHistory, sd.SafetyCrimeInformation, sd.EstimatedDailySpending = nil, nil, nil
			},
			want: func(sd *subdistrict.SubDistrict) {
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
			require.NoError(t, err)
			assert.Equal(t, want, stored)
		})
	}
}

func TestService_Update_ID(t *testing.T) {
	input := mentengInput()
	input.ID = 99 // the sub-district comes from the path, not from the replacement

	stored, err := updateSubDistrict(t, 7, input)
	require.NoError(t, err)
	assert.Equal(t, int64(7), stored.ID)
}

func TestService_Update_Rejected(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*subdistrict.SubDistrict)
		wantErr error
	}{
		{name: "name of spaces only", change: func(sd *subdistrict.SubDistrict) { sd.Name = "   " }, wantErr: subdistrict.ErrInvalidName},
		{name: "name of 101 characters", change: func(sd *subdistrict.SubDistrict) { sd.Name = strings.Repeat("a", 101) }, wantErr: subdistrict.ErrInvalidName},
		{name: "empty slug", change: func(sd *subdistrict.SubDistrict) { sd.Slug = "" }, wantErr: subdistrict.ErrInvalidSlug},
		{name: "slug of 101 characters", change: func(sd *subdistrict.SubDistrict) { sd.Slug = strings.Repeat("a", 101) }, wantErr: subdistrict.ErrInvalidSlug},
		{name: "negative estimated daily spending", change: func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(-0.01) },
			wantErr: subdistrict.ErrInvalidEstimatedDailySpending},
		{name: "estimated daily spending too large for DECIMAL(10,2)", change: func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(100000000.0) },
			wantErr: subdistrict.ErrInvalidEstimatedDailySpending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := mentengInput()
			tc.change(input)

			stored, err := updateSubDistrict(t, 7, input)
			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, stored, "rejected update reached storage")
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
		{name: "storage reports the sub-district is gone", updateErr: subdistrict.ErrSubDistrictNotFound, wantErr: subdistrict.ErrSubDistrictNotFound},
		{name: "new name or slug is already taken", updateErr: subdistrict.ErrSubDistrictExists, wantErr: subdistrict.ErrSubDistrictExists},
		{name: "storage failure is returned", updateErr: boom, wantErr: boom},
		{name: "stored sub-district is returned", updateErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&subdistrict.SubDistrict{ID: 7}, tc.updateErr)

			got, err := subdistrict.NewService(repo, noImages(t)).Update(context.Background(), 7, mentengInput())
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored sub-district was not returned")
				assert.Equal(t, int64(7), got.ID)
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
		{name: "deletes the sub-district", repoErr: nil, wantErr: nil},
		{name: "unknown sub-district", repoErr: subdistrict.ErrSubDistrictNotFound, wantErr: subdistrict.ErrSubDistrictNotFound},
		{name: "still referenced by places", repoErr: subdistrict.ErrSubDistrictInUse, wantErr: subdistrict.ErrSubDistrictInUse},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			// The repository queues the images, so the service deletes none.
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := subdistrict.NewService(repo, noImages(t)).Delete(context.Background(), 7)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_Delete_QueuesImages(t *testing.T) {
	// The repository queues every image for deletion, so the service must not
	// delete any of them itself.
	repo, images := newMocks(t)
	repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil)

	err := subdistrict.NewService(repo, images).Delete(context.Background(), 7)
	require.NoError(t, err, "the sub-district was deleted")
}

// -----------------------------------------------------------------------------
// Sub-district images
// -----------------------------------------------------------------------------

const (
	oldImageKey     = "subdistricts/7/image/9f8e7d6c-5b4a-4321-8fed-cba987654321.png"
	oldStaticMapKey = "subdistricts/7/static-map/1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d.png"
)

// slotCase is one of a sub-district's image slots, with its storage prefix
// and keys for sub-district 7.
type slotCase struct {
	slot      subdistrict.ImageSlot
	prefix    string
	uploadKey string
	newKey    string
	oldKey    string
	// setKey stores key as the image in this slot of sd.
	setKey func(sd *subdistrict.SubDistrict, key *string)
}

var slotCases = []slotCase{
	{
		slot:      subdistrict.SlotImage,
		prefix:    "subdistricts/7/image",
		uploadKey: "uploads/subdistricts/7/image/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
		newKey:    "subdistricts/7/image/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp",
		oldKey:    oldImageKey,
		setKey:    func(sd *subdistrict.SubDistrict, key *string) { sd.ImageKey = key },
	},
	{
		slot:      subdistrict.SlotStaticMap,
		prefix:    "subdistricts/7/static-map",
		uploadKey: "uploads/subdistricts/7/static-map/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.png",
		newKey:    "subdistricts/7/static-map/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.png",
		oldKey:    oldStaticMapKey,
		setKey:    func(sd *subdistrict.SubDistrict, key *string) { sd.StaticMapImageKey = key },
	},
}

func TestService_ImageUploadURL(t *testing.T) {
	for _, sc := range slotCases {
		t.Run(string(sc.slot), func(t *testing.T) {
			repo, images := newMocks(t)
			upload := storage.Upload{
				Key:             sc.uploadKey,
				PresignedUpload: storage.PresignedUpload{URL: "http://localhost:3900/signed", Method: "PUT"},
			}
			gomock.InOrder(
				repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				images.EXPECT().PresignUpload(gomock.Any(), sc.prefix, "image/webp").Return(upload, nil),
			)

			got, err := subdistrict.NewService(repo, images).ImageUploadURL(context.Background(), 7, sc.slot, "image/webp")
			require.NoError(t, err)
			assert.Equal(t, upload, got)
		})
	}
}

func TestService_ImageUploadURL_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown sub-district", getErr: subdistrict.ErrSubDistrictNotFound, wantErr: subdistrict.ErrSubDistrictNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be presigned for a sub-district that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := subdistrict.NewService(repo, noImages(t)).ImageUploadURL(context.Background(), 7, subdistrict.SlotImage, "image/webp")
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
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil)
			images.EXPECT().PresignUpload(gomock.Any(), "subdistricts/7/image", "image/gif").Return(storage.Upload{}, tc.presignErr)

			_, err := subdistrict.NewService(repo, images).ImageUploadURL(context.Background(), 7, subdistrict.SlotImage, "image/gif")
			require.ErrorIs(t, err, tc.presignErr)
		})
	}
}

func TestService_SetImage(t *testing.T) {
	tests := []struct {
		name   string
		hasOld bool
	}{
		{name: "first image", hasOld: false},
		{name: "replaces the previous image", hasOld: true},
	}

	for _, sc := range slotCases {
		for _, tc := range tests {
			t.Run(string(sc.slot)+"/"+tc.name, func(t *testing.T) {
				// The repository queues the previous image, and the image in the
				// other slot is never touched, so the service deletes nothing.
				repo, images := newMocks(t)
				existing := savedMenteng()
				if !tc.hasOld {
					sc.setKey(existing, nil)
				}
				updated := savedMenteng()
				sc.setKey(updated, ptr(sc.newKey))

				gomock.InOrder(
					repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(existing, nil),
					images.EXPECT().Promote(gomock.Any(), sc.prefix, sc.uploadKey).Return(sc.newKey, nil),
					repo.EXPECT().SetImageKey(gomock.Any(), int64(7), sc.slot, sc.newKey).Return(updated, nil),
				)

				got, err := subdistrict.NewService(repo, images).SetImage(context.Background(), 7, sc.slot, sc.uploadKey)
				require.NoError(t, err)
				assert.Same(t, updated, got, "want the stored sub-district")
			})
		}
	}
}

func TestService_SetImage_LookupFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{name: "unknown sub-district", getErr: subdistrict.ErrSubDistrictNotFound, wantErr: subdistrict.ErrSubDistrictNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be promoted for a sub-district that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := subdistrict.NewService(repo, noImages(t)).SetImage(context.Background(), 7, subdistrict.SlotImage, slotCases[0].uploadKey)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_SetImage_PromoteFailure(t *testing.T) {
	boom := errors.New("connection reset")
	sc := slotCases[0]

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
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil)
			images.EXPECT().Promote(gomock.Any(), sc.prefix, sc.uploadKey).Return("", tc.promoteErr)

			_, err := subdistrict.NewService(repo, images).SetImage(context.Background(), 7, sc.slot, sc.uploadKey)
			require.ErrorIs(t, err, tc.promoteErr)
		})
	}
}

func TestService_SetImage_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")
	sc := slotCases[0]

	tests := []struct {
		name      string
		setErr    error
		deleteErr error
	}{
		{name: "sub-district deleted meanwhile", setErr: subdistrict.ErrSubDistrictNotFound},
		{name: "storage failure is returned", setErr: boom},
		{name: "promoted image delete failure is ignored", setErr: boom, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The promoted image is removed, and the old image is kept.
			repo, images := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				images.EXPECT().Promote(gomock.Any(), sc.prefix, sc.uploadKey).Return(sc.newKey, nil),
				repo.EXPECT().SetImageKey(gomock.Any(), int64(7), sc.slot, sc.newKey).Return(nil, tc.setErr),
				images.EXPECT().Delete(gomock.Any(), sc.newKey).Return(tc.deleteErr),
			)

			_, err := subdistrict.NewService(repo, images).SetImage(context.Background(), 7, sc.slot, sc.uploadKey)
			require.ErrorIs(t, err, tc.setErr)
		})
	}
}
