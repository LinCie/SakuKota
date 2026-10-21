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

	_, err := subdistrict.NewService(repo).Create(context.Background(), sd)
	return stored, err
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
		{name: "static map image URL of 255 characters", in: subdistrict.SubDistrict{StaticMapImageURL: ptr(strings.Repeat("a", 255))}},
		{name: "static map image URL of 256 characters", in: subdistrict.SubDistrict{StaticMapImageURL: ptr(strings.Repeat("a", 256))},
			wantErr: subdistrict.ErrInvalidStaticMapImageURL},
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
			StaticMapImageURL:      ptr("https://example.com/menteng.png"),
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

			got, err := subdistrict.NewService(repo).Create(context.Background(), &subdistrict.SubDistrict{
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
// with every optional field set.
func savedMenteng() *subdistrict.SubDistrict {
	return &subdistrict.SubDistrict{
		ID:                     7,
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           ptr("Banjir 2020"),
		SafetyCrimeInformation: ptr("Aman di siang hari"),
		EstimatedDailySpending: ptr(1500000.5),
		StaticMapImageURL:      ptr("https://example.com/menteng.png"),
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

	_, err := subdistrict.NewService(repo).Update(context.Background(), id, patch)
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

			got, err := subdistrict.NewService(repo).GetSubDistrict(context.Background(), 7)
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

			_, err := subdistrict.NewService(repo).Update(context.Background(), 7,
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
		{name: "static map image URL is replaced", patch: subdistrict.SubDistrictPatch{StaticMapImageURL: present("https://example.com/menteng-2.png")},
			change: func(sd *subdistrict.SubDistrict) { sd.StaticMapImageURL = ptr("https://example.com/menteng-2.png") }},
		{name: "null clears every nullable field", patch: subdistrict.SubDistrictPatch{
			FloodHistory:           null[string](),
			SafetyCrimeInformation: null[string](),
			EstimatedDailySpending: null[float64](),
			StaticMapImageURL:      null[string](),
		}, change: func(sd *subdistrict.SubDistrict) {
			sd.FloodHistory, sd.SafetyCrimeInformation = nil, nil
			sd.EstimatedDailySpending, sd.StaticMapImageURL = nil, nil
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
		{name: "static map image URL of 256 characters", patch: subdistrict.SubDistrictPatch{StaticMapImageURL: present(strings.Repeat("a", 256))},
			wantErr: subdistrict.ErrInvalidStaticMapImageURL},
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

			got, err := subdistrict.NewService(repo).Update(context.Background(), 7,
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
			repo.EXPECT().Delete(gomock.Any(), int64(7)).Return(tc.repoErr)

			err := subdistrict.NewService(repo).Delete(context.Background(), 7)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
