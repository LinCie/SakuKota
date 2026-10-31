package subdistrict_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"rantau-go/internal/httpx"
	"rantau-go/internal/module/subdistrict"
	"rantau-go/internal/module/subdistrict/mock"
	"rantau-go/internal/storage"
)

// family is one emoji made of five code points: man, ZWJ, woman, ZWJ, girl.
const family = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

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

func TestService_CreateSubDistrict_Name(t *testing.T) {
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
			if stored == nil {
				t.Fatal("accepted sub-district was not stored")
			}
			if stored.Name != tc.wantName {
				t.Errorf("stored name = %q, want %q", stored.Name, tc.wantName)
			}
		})
	}
}

func TestService_CreateSubDistrict_Slug(t *testing.T) {
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
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatalf("invalid slug reached storage as %q", stored.Slug)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored == nil {
				t.Fatal("accepted sub-district was not stored")
			}
			if stored.Slug != tc.wantSlug {
				t.Errorf("stored slug = %q, want %q", stored.Slug, tc.wantSlug)
			}
		})
	}
}

func TestService_CreateSubDistrict_OptionalFieldLimits(t *testing.T) {
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
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if stored != nil {
					t.Fatal("rejected sub-district reached storage")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored == nil {
				t.Fatal("accepted sub-district was not stored")
			}
		})
	}
}

func TestService_CreateSubDistrict_StoresFieldsAsSent(t *testing.T) {
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored == nil {
				t.Fatal("accepted sub-district was not stored")
			}
			if !reflect.DeepEqual(*stored, tc.in) {
				t.Errorf("stored = %+v, want %+v", *stored, tc.in)
			}
		})
	}
}

func TestService_CreateSubDistrict_Storage(t *testing.T) {
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
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (got == nil || got.ID != 7) {
				t.Errorf("returned sub-district = %+v, want the stored sub-district with ID 7", got)
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

// updateSubDistrict runs Service.Update for sub-district id, where existing is
// the stored sub-district. It returns the sub-district the repository was asked
// to store, or nil if storage was never reached.
func updateSubDistrict(t *testing.T, existing *subdistrict.SubDistrict, id int64, patch subdistrict.SubDistrictPatch) (*subdistrict.SubDistrict, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))
	repo.EXPECT().GetSubDistrict(gomock.Any(), id).Return(existing, nil).AnyTimes()

	var stored *subdistrict.SubDistrict
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *subdistrict.SubDistrict) (*subdistrict.SubDistrict, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := subdistrict.NewService(repo, noImages(t)).Update(context.Background(), id, patch)
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
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (got == nil || got.ID != 7) {
				t.Errorf("returned sub-district = %+v, want the sub-district with ID 7", got)
			}
		})
	}
}

func TestService_UpdateSubDistrict_LookupFailure(t *testing.T) {
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
			// No Update expectation: nothing may be stored when the sub-district cannot be read.
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(nil, tc.getErr)

			_, err := subdistrict.NewService(repo, noImages(t)).Update(context.Background(), 7,
				subdistrict.SubDistrictPatch{Name: present("Kemang")})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestService_UpdateSubDistrict_Fields(t *testing.T) {
	tests := []struct {
		name   string
		patch  subdistrict.SubDistrictPatch
		change func(*subdistrict.SubDistrict) // turns savedMenteng into the wanted stored sub-district
	}{
		{name: "empty patch changes nothing", patch: subdistrict.SubDistrictPatch{}, change: func(*subdistrict.SubDistrict) {}},
		{name: "name is trimmed", patch: subdistrict.SubDistrictPatch{Name: present("  Menteng Atas\t")},
			change: func(sd *subdistrict.SubDistrict) { sd.Name = "Menteng Atas" }},
		{name: "name of 100 characters is allowed", patch: subdistrict.SubDistrictPatch{Name: present(strings.Repeat("a", 100))},
			change: func(sd *subdistrict.SubDistrict) { sd.Name = strings.Repeat("a", 100) }},
		{name: "slug is trimmed", patch: subdistrict.SubDistrictPatch{Slug: present("  menteng-atas ")},
			change: func(sd *subdistrict.SubDistrict) { sd.Slug = "menteng-atas" }},
		{name: "flood history is replaced", patch: subdistrict.SubDistrictPatch{FloodHistory: present("Banjir 2024")},
			change: func(sd *subdistrict.SubDistrict) { sd.FloodHistory = ptr("Banjir 2024") }},
		{name: "safety information is replaced", patch: subdistrict.SubDistrictPatch{SafetyCrimeInformation: present("Rawan malam")},
			change: func(sd *subdistrict.SubDistrict) { sd.SafetyCrimeInformation = ptr("Rawan malam") }},
		{name: "estimated daily spending is replaced", patch: subdistrict.SubDistrictPatch{EstimatedDailySpending: present(750000.25)},
			change: func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(750000.25) }},
		{name: "estimated daily spending of zero is allowed", patch: subdistrict.SubDistrictPatch{EstimatedDailySpending: present(0.0)},
			change: func(sd *subdistrict.SubDistrict) { sd.EstimatedDailySpending = ptr(0.0) }},
		{name: "null clears every nullable field", patch: subdistrict.SubDistrictPatch{
			FloodHistory:           null[string](),
			SafetyCrimeInformation: null[string](),
			EstimatedDailySpending: null[float64](),
		}, change: func(sd *subdistrict.SubDistrict) {
			sd.FloodHistory, sd.SafetyCrimeInformation, sd.EstimatedDailySpending = nil, nil, nil
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := savedMenteng()
			tc.change(want)

			stored, err := updateSubDistrict(t, savedMenteng(), 7, tc.patch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(stored, want) {
				t.Errorf("stored = %+v, want %+v", stored, want)
			}
		})
	}
}

func TestService_UpdateSubDistrict_Rejected(t *testing.T) {
	tests := []struct {
		name    string
		patch   subdistrict.SubDistrictPatch
		wantErr error
	}{
		{name: "name of spaces only", patch: subdistrict.SubDistrictPatch{Name: present("   ")}, wantErr: subdistrict.ErrInvalidName},
		{name: "name of 101 characters", patch: subdistrict.SubDistrictPatch{Name: present(strings.Repeat("a", 101))}, wantErr: subdistrict.ErrInvalidName},
		{name: "empty slug", patch: subdistrict.SubDistrictPatch{Slug: present("")}, wantErr: subdistrict.ErrInvalidSlug},
		{name: "slug of 101 characters", patch: subdistrict.SubDistrictPatch{Slug: present(strings.Repeat("a", 101))}, wantErr: subdistrict.ErrInvalidSlug},
		{name: "negative estimated daily spending", patch: subdistrict.SubDistrictPatch{EstimatedDailySpending: present(-0.01)},
			wantErr: subdistrict.ErrInvalidEstimatedDailySpending},
		{name: "estimated daily spending too large for DECIMAL(10,2)", patch: subdistrict.SubDistrictPatch{EstimatedDailySpending: present(100000000.0)},
			wantErr: subdistrict.ErrInvalidEstimatedDailySpending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := updateSubDistrict(t, savedMenteng(), 7, tc.patch)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if stored != nil {
				t.Fatalf("rejected update reached storage as %+v", *stored)
			}
		})
	}
}

func TestService_UpdateSubDistrict_Storage(t *testing.T) {
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
			repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(savedMenteng(), nil)
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&subdistrict.SubDistrict{ID: 7}, tc.updateErr)

			got, err := subdistrict.NewService(repo, noImages(t)).Update(context.Background(), 7,
				subdistrict.SubDistrictPatch{Name: present("Menteng Atas")})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (got == nil || got.ID != 7) {
				t.Errorf("returned sub-district = %+v, want the stored sub-district with ID 7", got)
			}
		})
	}
}

func TestService_DeleteSubDistrict(t *testing.T) {
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
			// The sub-district has no images, so no image may be deleted.
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(nil, tc.repoErr)

			err := subdistrict.NewService(repo, noImages(t)).Delete(context.Background(), 7)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestService_DeleteSubDistrict_Images(t *testing.T) {
	tests := []struct {
		name      string
		deleteErr error
	}{
		{name: "deletes every image", deleteErr: nil},
		{name: "image delete failure is ignored", deleteErr: errors.New("connection reset")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, images := newMocks(t)
			gomock.InOrder(
				repo.EXPECT().Delete(gomock.Any(), int64(7)).Return([]string{oldImageKey, oldStaticMapKey}, nil),
				images.EXPECT().Delete(gomock.Any(), oldImageKey).Return(tc.deleteErr),
				images.EXPECT().Delete(gomock.Any(), oldStaticMapKey).Return(nil),
			)

			err := subdistrict.NewService(repo, images).Delete(context.Background(), 7)
			if err != nil {
				t.Fatalf("err = %v, want nil: the sub-district was deleted", err)
			}
		})
	}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, upload) {
				t.Errorf("upload = %+v, want %+v", got, upload)
			}
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
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
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
			if !errors.Is(err, tc.presignErr) {
				t.Fatalf("err = %v, want %v", err, tc.presignErr)
			}
		})
	}
}

func TestService_SetImage(t *testing.T) {
	tests := []struct {
		name      string
		hasOld    bool
		deleteErr error
	}{
		{name: "first image", hasOld: false},
		{name: "replaces the previous image", hasOld: true},
		{name: "previous image delete failure is ignored", hasOld: true, deleteErr: errors.New("connection reset")},
	}

	for _, sc := range slotCases {
		for _, tc := range tests {
			t.Run(string(sc.slot)+"/"+tc.name, func(t *testing.T) {
				// The image in the other slot is never deleted.
				repo, images := newMocks(t)
				existing := savedMenteng()
				if !tc.hasOld {
					sc.setKey(existing, nil)
				}
				updated := savedMenteng()
				sc.setKey(updated, ptr(sc.newKey))

				calls := []any{
					repo.EXPECT().GetSubDistrict(gomock.Any(), int64(7)).Return(existing, nil),
					images.EXPECT().Promote(gomock.Any(), sc.prefix, sc.uploadKey).Return(sc.newKey, nil),
					repo.EXPECT().SetImageKey(gomock.Any(), int64(7), sc.slot, sc.newKey).Return(updated, nil),
				}
				if tc.hasOld {
					calls = append(calls, images.EXPECT().Delete(gomock.Any(), sc.oldKey).Return(tc.deleteErr))
				}
				gomock.InOrder(calls...)

				got, err := subdistrict.NewService(repo, images).SetImage(context.Background(), 7, sc.slot, sc.uploadKey)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != updated {
					t.Errorf("got %+v, want the stored sub-district %+v", got, updated)
				}
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
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
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
			if !errors.Is(err, tc.promoteErr) {
				t.Fatalf("err = %v, want %v", err, tc.promoteErr)
			}
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
			if !errors.Is(err, tc.setErr) {
				t.Fatalf("err = %v, want %v", err, tc.setErr)
			}
		})
	}
}
