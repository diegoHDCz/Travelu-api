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

func (f *fakeRepository) Update(_ context.Context, r Review) (Review, error) {
	if _, ok := f.byID[r.ID]; !ok {
		return Review{}, ErrNotFound
	}
	f.byID[r.ID] = r

	reviews := f.byListing[r.ListingID]
	for i, existing := range reviews {
		if existing.ID == r.ID {
			reviews[i] = r
			break
		}
	}

	return r, nil
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

func TestService_Update(t *testing.T) {
	svc := NewService(newFakeRepository())
	customerID := uuid.NewString()

	created, err := svc.Create(context.Background(), CreateInput{
		CustomerID: customerID,
		ListingID:  uuid.NewString(),
		Rating:     3,
	})
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), UpdateInput{
		ID:         created.ID,
		CustomerID: customerID,
		Rating:     5,
		Title:      "Great stay",
		Comment:    "Loved it",
	})

	require.NoError(t, err)
	assert.Equal(t, 5, updated.Rating)
	assert.Equal(t, "Great stay", updated.Title)
	assert.Equal(t, "Loved it", updated.Comment)
}

func TestService_Update_WrongCustomer(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		CustomerID: uuid.NewString(),
		ListingID:  uuid.NewString(),
		Rating:     3,
	})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), UpdateInput{
		ID:         created.ID,
		CustomerID: uuid.NewString(),
		Rating:     1,
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.Update(context.Background(), UpdateInput{
		ID:         "missing",
		CustomerID: uuid.NewString(),
		Rating:     1,
	})
	assert.ErrorIs(t, err, ErrNotFound)
}
