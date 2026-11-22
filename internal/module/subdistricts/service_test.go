package subdistricts_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/subdistricts"
	"rantau-go/internal/module/subdistricts/mock"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
)

// family is one emoji made of five code points: man, ZWJ, woman, ZWJ, girl.
const family = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

// createSubDistrict runs Service.Create for sd. It returns the sub-district the
// repository was asked to store, or nil if storage was never reached.
func createSubDistrict(t *testing.T, sd *subdistricts.SubDistrict) (*subdistricts.SubDistrict, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	var stored *subdistricts.SubDistrict
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *subdistricts.SubDistrict) (*subdistricts.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := subdistricts.NewService(repo, noMedia(t)).Create(context.Background(), sd)
	return stored, err
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
		{name: "empty", input: "", wantErr: subdistricts.ErrInvalidName},
		{name: "spaces only", input: "   ", wantErr: subdistricts.ErrInvalidName},
		{name: "100 characters", input: strings.Repeat("a", 100), wantName: strings.Repeat("a", 100)},
		{name: "101 characters", input: strings.Repeat("a", 101), wantErr: subdistricts.ErrInvalidName},
		{name: "100 multibyte characters", input: strings.Repeat("é", 100), wantName: strings.Repeat("é", 100)},
		{name: "101 multibyte characters", input: strings.Repeat("é", 101), wantErr: subdistricts.ErrInvalidName},
		{name: "20 family emoji are 100 code points", input: strings.Repeat(family, 20), wantName: strings.Repeat(family, 20)},
		{name: "21 family emoji are 105 code points", input: strings.Repeat(family, 21), wantErr: subdistricts.ErrInvalidName},
		{name: "length is counted after trimming", input: "  " + strings.Repeat("a", 100) + "  ", wantName: strings.Repeat("a", 100)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createSubDistrict(t, &subdistricts.SubDistrict{Name: tc.input, Slug: "menteng"})
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
		{name: "101 characters", input: strings.Repeat("a", 101), wantErr: subdistricts.ErrInvalidSlug},
		{name: "empty", input: "", wantErr: subdistricts.ErrInvalidSlug},
		{name: "spaces only", input: "   ", wantErr: subdistricts.ErrInvalidSlug},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createSubDistrict(t, &subdistricts.SubDistrict{Name: "Menteng", Slug: tc.input})
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
		in      subdistricts.SubDistrict
		wantErr error // nil means the sub-district is accepted
	}{
		{name: "estimated daily spending of zero", in: subdistricts.SubDistrict{EstimatedDailySpending: new(0.0)}},
		{name: "estimated daily spending with cents", in: subdistricts.SubDistrict{EstimatedDailySpending: new(1500000.5)}},
		{name: "largest estimated daily spending", in: subdistricts.SubDistrict{EstimatedDailySpending: new(99999999.99)}},
		{name: "negative estimated daily spending", in: subdistricts.SubDistrict{EstimatedDailySpending: new(-0.01)},
			wantErr: subdistricts.ErrInvalidEstimatedDailySpending},
		{name: "estimated daily spending too large for DECIMAL(10,2)", in: subdistricts.SubDistrict{EstimatedDailySpending: new(100000000.0)},
			wantErr: subdistricts.ErrInvalidEstimatedDailySpending},
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
		set  func(*subdistricts.SubDistrict, *string)
		get  func(*subdistricts.SubDistrict) *string
	}{
		{name: "flood history",
			set: func(sd *subdistricts.SubDistrict, v *string) { sd.FloodHistory = v },
			get: func(sd *subdistricts.SubDistrict) *string { return sd.FloodHistory }},
		{name: "safety and crime information",
			set: func(sd *subdistricts.SubDistrict, v *string) { sd.SafetyCrimeInformation = v },
			get: func(sd *subdistricts.SubDistrict) *string { return sd.SafetyCrimeInformation }},
	}

	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{name: "surrounding whitespace is removed", input: "  Banjir 2020\n", want: new("Banjir 2020")},
		{name: "inner spacing is kept", input: "Banjir  2020", want: new("Banjir  2020")},
		{name: "blank is stored as absent", input: " \t\n ", want: nil},
	}

	for _, f := range fields {
		for _, tc := range tests {
			t.Run("create/"+f.name+"/"+tc.name, func(t *testing.T) {
				sd := &subdistricts.SubDistrict{Name: "Menteng", Slug: "menteng"}
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
		in   subdistricts.SubDistrict
	}{
		{name: "every field set", in: subdistricts.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           new("Banjir tahunan"),
			SafetyCrimeInformation: new("Aman di siang hari"),
			EstimatedDailySpending: new(1500000.5),
		}},
		{name: "only name and slug", in: subdistricts.SubDistrict{Name: "Menteng", Slug: "menteng"}},
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
		{name: "duplicate name or slug is returned", storeErr: subdistricts.ErrSubDistrictExists, wantErr: subdistricts.ErrSubDistrictExists},
		{name: "storage failure is returned", storeErr: boom, wantErr: boom},
		{name: "stored sub-district is returned", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&subdistricts.SubDistrict{ID: 7}, tc.storeErr)

			got, err := subdistricts.NewService(repo, noMedia(t)).Create(context.Background(), &subdistricts.SubDistrict{
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
// with every optional field and a media set.
func savedMenteng() *subdistricts.SubDistrict {
	return &subdistricts.SubDistrict{
		ID:                     7,
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           new("Banjir 2020"),
		SafetyCrimeInformation: new("Aman di siang hari"),
		EstimatedDailySpending: new(1500000.5),
		Medias: []storage.Media{
			{ID: 3, Key: "subdistricts/7/medias/9f8e7d6c-5b4a-4321-8fed-cba987654321.png", ContentType: "image/png", Size: 1024, Position: 1},
		},
	}
}

// mentengInput is a full replacement for savedMenteng: a Menteng with every
// optional field set. Updates change one field of it at a time.
func mentengInput() *subdistricts.SubDistrict {
	return &subdistricts.SubDistrict{
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
func updateSubDistrict(t *testing.T, id int64, sd *subdistricts.SubDistrict) (*subdistricts.SubDistrict, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))

	var stored *subdistricts.SubDistrict
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *subdistricts.SubDistrict) (*subdistricts.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := subdistricts.NewService(repo, noMedia(t)).Update(context.Background(), id, sd)
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
		{name: "unknown sub-district", repoErr: subdistricts.ErrSubDistrictNotFound, wantErr: subdistricts.ErrSubDistrictNotFound},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var found *subdistricts.SubDistrict
			if tc.repoErr == nil {
				found = savedMenteng()
			}

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(found, tc.repoErr)

			got, err := subdistricts.NewService(repo, noMedia(t)).GetSubDistrict(context.Background(), 7)
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
		change func(*subdistricts.SubDistrict) // turns mentengInput into the replacement
		want   func(*subdistricts.SubDistrict) // turns mentengInput into the wanted stored sub-district
	}{
		{name: "same fields change nothing", change: func(*subdistricts.SubDistrict) {}, want: func(*subdistricts.SubDistrict) {}},
		{name: "name is trimmed",
			change: func(sd *subdistricts.SubDistrict) { sd.Name = "  Menteng Atas\t" },
			want:   func(sd *subdistricts.SubDistrict) { sd.Name = "Menteng Atas" }},
		{name: "name of 100 characters is allowed",
			change: func(sd *subdistricts.SubDistrict) { sd.Name = strings.Repeat("a", 100) },
			want:   func(sd *subdistricts.SubDistrict) { sd.Name = strings.Repeat("a", 100) }},
		{name: "slug is trimmed",
			change: func(sd *subdistricts.SubDistrict) { sd.Slug = "  menteng-atas " },
			want:   func(sd *subdistricts.SubDistrict) { sd.Slug = "menteng-atas" }},
		{name: "flood history is replaced",
			change: func(sd *subdistricts.SubDistrict) { sd.FloodHistory = new("Banjir 2024") },
			want:   func(sd *subdistricts.SubDistrict) { sd.FloodHistory = new("Banjir 2024") }},
		{name: "safety information is replaced",
			change: func(sd *subdistricts.SubDistrict) { sd.SafetyCrimeInformation = new("Rawan malam") },
			want:   func(sd *subdistricts.SubDistrict) { sd.SafetyCrimeInformation = new("Rawan malam") }},
		{name: "estimated daily spending is replaced",
			change: func(sd *subdistricts.SubDistrict) { sd.EstimatedDailySpending = new(750000.25) },
			want:   func(sd *subdistricts.SubDistrict) { sd.EstimatedDailySpending = new(750000.25) }},
		{name: "estimated daily spending of zero is allowed",
			change: func(sd *subdistricts.SubDistrict) { sd.EstimatedDailySpending = new(0.0) },
			want:   func(sd *subdistricts.SubDistrict) { sd.EstimatedDailySpending = new(0.0) }},
		{name: "absent nullable fields are cleared",
			change: func(sd *subdistricts.SubDistrict) {
				sd.FloodHistory, sd.SafetyCrimeInformation, sd.EstimatedDailySpending = nil, nil, nil
			},
			want: func(sd *subdistricts.SubDistrict) {
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
		change  func(*subdistricts.SubDistrict)
		wantErr error
	}{
		{name: "name of spaces only", change: func(sd *subdistricts.SubDistrict) { sd.Name = "   " }, wantErr: subdistricts.ErrInvalidName},
		{name: "name of 101 characters", change: func(sd *subdistricts.SubDistrict) { sd.Name = strings.Repeat("a", 101) }, wantErr: subdistricts.ErrInvalidName},
		{name: "empty slug", change: func(sd *subdistricts.SubDistrict) { sd.Slug = "" }, wantErr: subdistricts.ErrInvalidSlug},
		{name: "slug of 101 characters", change: func(sd *subdistricts.SubDistrict) { sd.Slug = strings.Repeat("a", 101) }, wantErr: subdistricts.ErrInvalidSlug},
		{name: "negative estimated daily spending", change: func(sd *subdistricts.SubDistrict) { sd.EstimatedDailySpending = new(-0.01) },
			wantErr: subdistricts.ErrInvalidEstimatedDailySpending},
		{name: "estimated daily spending too large for DECIMAL(10,2)", change: func(sd *subdistricts.SubDistrict) { sd.EstimatedDailySpending = new(100000000.0) },
			wantErr: subdistricts.ErrInvalidEstimatedDailySpending},
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
		{name: "storage reports the sub-district is gone", updateErr: subdistricts.ErrSubDistrictNotFound, wantErr: subdistricts.ErrSubDistrictNotFound},
		{name: "new name or slug is already taken", updateErr: subdistricts.ErrSubDistrictExists, wantErr: subdistricts.ErrSubDistrictExists},
		{name: "storage failure is returned", updateErr: boom, wantErr: boom},
		{name: "stored sub-district is returned", updateErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&subdistricts.SubDistrict{ID: 7}, tc.updateErr)

			got, err := subdistricts.NewService(repo, noMedia(t)).Update(context.Background(), 7, mentengInput())
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
		{name: "unknown sub-district", repoErr: subdistricts.ErrSubDistrictNotFound, wantErr: subdistricts.ErrSubDistrictNotFound},
		{name: "still referenced by places", repoErr: subdistricts.ErrSubDistrictInUse, wantErr: subdistricts.ErrSubDistrictInUse},
		{name: "storage failure is returned", repoErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			// The repository queues the images, so the service deletes none.
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := subdistricts.NewService(repo, noMedia(t)).Delete(context.Background(), 7)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestService_Delete_QueuesImages(t *testing.T) {
	// The repository queues every image for deletion, so the service must not
	// delete any of them itself.
	repo, media := newMocks(t)
	repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil)

	err := subdistricts.NewService(repo, media).Delete(context.Background(), 7)
	require.NoError(t, err, "the sub-district was deleted")
}

// -----------------------------------------------------------------------------
// Sub-district images
// -----------------------------------------------------------------------------

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
			repo, media := newMocks(t)
			upload := storage.Upload{
				Key:    sc.uploadKey,
				URL:    "http://localhost:3900/signed",
				Method: "PUT",
			}
			gomock.InOrder(
				repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				media.EXPECT().PresignUpload(gomock.Any(), sc.prefix, "image/webp").Return(upload, nil),
			)

			got, err := subdistricts.NewService(repo, media).MediaUploadURL(context.Background(), 7, sc.slot, "image/webp")
			require.NoError(t, err)
			assert.Equal(t, upload, got)
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
		{name: "unknown sub-district", getErr: subdistricts.ErrSubDistrictNotFound, wantErr: subdistricts.ErrSubDistrictNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be presigned for a sub-district that cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := subdistricts.NewService(repo, noMedia(t)).MediaUploadURL(context.Background(), 7, subdistricts.SlotMedias, "image/webp")
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
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil)
			media.EXPECT().PresignUpload(gomock.Any(), "subdistricts/7/medias", "image/gif").Return(storage.Upload{}, tc.presignErr)

			_, err := subdistricts.NewService(repo, media).MediaUploadURL(context.Background(), 7, subdistricts.SlotMedias, "image/gif")
			require.ErrorIs(t, err, tc.presignErr)
		})
	}
}

// storeCase is an operation that promotes an upload and stores the image.
type storeCase struct {
	name string
	sc   slotCase
	// expectStore expects the repository to store m for sub-district 7.
	expectStore func(repo *mock.MockRepository, m storage.Media) *gomock.Call
	run         func(s subdistricts.Service, uploadKey string) (*subdistricts.SubDistrict, error)
}

var storeCases = []storeCase{
	{
		name: "add media",
		sc:   mediasSlot,
		expectStore: func(repo *mock.MockRepository, m storage.Media) *gomock.Call {
			return repo.EXPECT().AddMedia(gomock.Any(), int64(7), m)
		},
		run: func(s subdistricts.Service, uploadKey string) (*subdistricts.SubDistrict, error) {
			return s.AddMedia(context.Background(), 7, uploadKey)
		},
	},
}

// promotedMedia is the media Promote returns for an upload stored under key.
func promotedMedia(key string) storage.Media {
	return storage.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

func TestService_StoreMedia(t *testing.T) {
	for _, op := range storeCases {
		t.Run(op.name, func(t *testing.T) {
			// The repository queues any replaced image, so the service deletes
			// nothing.
			repo, media := newMocks(t)
			promoted := promotedMedia(op.sc.newKey)
			updated := savedMenteng()

			gomock.InOrder(
				repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
				media.EXPECT().Promote(gomock.Any(), op.sc.prefix, op.sc.uploadKey).Return(promoted, nil),
				op.expectStore(repo, promoted).Return(updated, nil),
			)

			got, err := op.run(subdistricts.NewService(repo, media), op.sc.uploadKey)
			require.NoError(t, err)
			assert.Same(t, updated, got, "want the stored sub-district")
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
		{name: "unknown sub-district", getErr: subdistricts.ErrSubDistrictNotFound, wantErr: subdistricts.ErrSubDistrictNotFound},
		{name: "lookup failure is returned", getErr: boom, wantErr: boom},
	}

	for _, op := range storeCases {
		for _, tc := range tests {
			t.Run(op.name+"/"+tc.name, func(t *testing.T) {
				// Nothing may be promoted for a sub-district that cannot be read.
				repo := mock.NewMockRepository(gomock.NewController(t))
				repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(nil, tc.getErr)

				_, err := op.run(subdistricts.NewService(repo, noMedia(t)), op.sc.uploadKey)
				require.ErrorIs(t, err, tc.wantErr)
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
		{name: "invalid upload key", promoteErr: storage.ErrInvalidUploadKey},
		{name: "image not uploaded", promoteErr: storage.ErrMediaNotUploaded},
		{name: "invalid image type", promoteErr: storage.ErrInvalidMediaType},
		{name: "image too large", promoteErr: storage.ErrImageTooLarge},
		{name: "storage failure is returned", promoteErr: boom},
	}

	for _, op := range storeCases {
		for _, tc := range tests {
			t.Run(op.name+"/"+tc.name, func(t *testing.T) {
				// Nothing may be stored.
				repo, media := newMocks(t)
				repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil)
				media.EXPECT().Promote(gomock.Any(), op.sc.prefix, op.sc.uploadKey).Return(storage.Media{}, tc.promoteErr)

				_, err := op.run(subdistricts.NewService(repo, media), op.sc.uploadKey)
				require.ErrorIs(t, err, tc.promoteErr)
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
		{name: "sub-district deleted meanwhile", storeErr: subdistricts.ErrSubDistrictNotFound},
		{name: "media limit reached", storeErr: storage.ErrMediaLimitReached},
		{name: "storage failure is returned", storeErr: boom},
		{name: "promoted image delete failure is ignored", storeErr: boom, deleteErr: errors.New("timeout")},
	}

	for _, op := range storeCases {
		for _, tc := range tests {
			t.Run(op.name+"/"+tc.name, func(t *testing.T) {
				// The promoted image is removed.
				repo, media := newMocks(t)
				promoted := promotedMedia(op.sc.newKey)
				gomock.InOrder(
					repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil),
					media.EXPECT().Promote(gomock.Any(), op.sc.prefix, op.sc.uploadKey).Return(promoted, nil),
					op.expectStore(repo, promoted).Return(nil, tc.storeErr),
					media.EXPECT().Delete(gomock.Any(), op.sc.newKey).Return(tc.deleteErr),
				)

				_, err := op.run(subdistricts.NewService(repo, media), op.sc.uploadKey)
				require.ErrorIs(t, err, tc.storeErr)
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

			err := subdistricts.NewService(repo, noMedia(t)).DeleteMedia(context.Background(), 7, 3)
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

// -----------------------------------------------------------------------------
// Sub-district boundary
// -----------------------------------------------------------------------------

func TestService_SetBoundary(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "stores the boundary", repoErr: nil},
		{name: "unknown sub-district", repoErr: subdistricts.ErrSubDistrictNotFound},
		{name: "boundary crosses itself", repoErr: subdistricts.ErrInvalidBoundary},
		{name: "storage failure is returned", repoErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boundary := mustBoundary(t, squareJSON)
			var stored *subdistricts.SubDistrict
			if tc.repoErr == nil {
				stored = savedMenteng()
			}

			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().SetBoundary(gomock.Any(), int64(7), boundary).Return(stored, tc.repoErr)

			got, err := subdistricts.NewService(repo, noMedia(t)).SetBoundary(context.Background(), 7, boundary)
			require.ErrorIs(t, err, tc.repoErr)
			assert.Same(t, stored, got)
		})
	}
}

func TestService_SetBoundary_Rejected(t *testing.T) {
	tests := []struct {
		name     string
		boundary *postgis.MultiPolygon
	}{
		{name: "no boundary", boundary: nil},
		{name: "a vertex east of Indonesia", boundary: mustBoundary(t,
			`{"type":"Polygon","coordinates":[[[106.8,-6.2],[141.01,-6.2],[106.9,-6.3],[106.8,-6.2]]]}`)},
		{name: "one polygon outside Indonesia", boundary: mustBoundary(t,
			`{"type":"MultiPolygon","coordinates":[[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.2]]],`+
				`[[[0,0],[1,0],[1,1],[0,0]]]]}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: a rejected boundary must never reach storage.
			repo := mock.NewMockRepository(gomock.NewController(t))

			got, err := subdistricts.NewService(repo, noMedia(t)).SetBoundary(context.Background(), 7, tc.boundary)
			require.ErrorIs(t, err, subdistricts.ErrInvalidBoundary)
			assert.Nil(t, got)
		})
	}
}

func TestService_SetBoundary_EdgesOfIndonesia(t *testing.T) {
	// The box's edges count as inside.
	boundary := mustBoundary(t, `{"type":"Polygon","coordinates":[[[95,-11],[141,-11],[141,6],[95,6],[95,-11]]]}`)

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().SetBoundary(gomock.Any(), int64(7), boundary).Return(savedMenteng(), nil)

	_, err := subdistricts.NewService(repo, noMedia(t)).SetBoundary(context.Background(), 7, boundary)
	require.NoError(t, err)
}
