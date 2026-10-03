package review

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository used by service unit tests.
type fakeRepository struct {
	byID      map[string]Review
	byListing map[string][]Review
	nextID    int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]Review{}, byListing: map[string][]Review{}}
}

func (f *fakeRepository) Create(_ context.Context, r Review) (Review, error) {
	f.nextID++
	r.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[r.ID] = r
	f.byListing[r.ListingID] = append(f.byListing[r.ListingID], r)
	return r, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (Review, error) {
	r, ok := f.byID[id]
	if !ok {
		return Review{}, ErrNotFound
	}
	return r, nil
}

func (f *fakeRepository) ListByListingID(_ context.Context, listingID string) ([]Review, error) {
	return f.byListing[listingID], nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		CustomerID: uuid.NewString(),
		ListingID:  uuid.NewString(),
		Rating:     5,
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.False(t, created.IsVerified)
	assert.True(t, created.IsApproved)
}

func TestService_ListByListingID(t *testing.T) {
	svc := NewService(newFakeRepository())
	listingID := uuid.NewString()

	_, err := svc.Create(context.Background(), CreateInput{CustomerID: uuid.NewString(), ListingID: listingID, Rating: 4})
	require.NoError(t, err)

	reviews, err := svc.ListByListingID(context.Background(), listingID)
	require.NoError(t, err)
	assert.Len(t, reviews, 1)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}
