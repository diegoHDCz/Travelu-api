package tripdate

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository used by service unit tests.
type fakeRepository struct {
	byID      map[string]TripDate
	byListing map[string][]TripDate
	nextID    int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]TripDate{}, byListing: map[string][]TripDate{}}
}

func (f *fakeRepository) Create(_ context.Context, td TripDate) (TripDate, error) {
	f.nextID++
	td.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[td.ID] = td
	f.byListing[td.ListingID] = append(f.byListing[td.ListingID], td)
	return td, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (TripDate, error) {
	td, ok := f.byID[id]
	if !ok {
		return TripDate{}, ErrNotFound
	}
	return td, nil
}

func (f *fakeRepository) ListByListingID(_ context.Context, listingID string) ([]TripDate, error) {
	return f.byListing[listingID], nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())
	listingID := uuid.NewString()
	start := time.Now().UTC()

	created, err := svc.Create(context.Background(), CreateInput{
		ListingID: listingID,
		StartDate: start,
		EndDate:   start.Add(48 * time.Hour),
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.True(t, created.IsActive)
	assert.Zero(t, created.CurrentBookings)
}

func TestService_ListByListingID(t *testing.T) {
	svc := NewService(newFakeRepository())
	listingID := uuid.NewString()
	start := time.Now().UTC()

	_, err := svc.Create(context.Background(), CreateInput{ListingID: listingID, StartDate: start, EndDate: start.Add(time.Hour)})
	require.NoError(t, err)

	tripDates, err := svc.ListByListingID(context.Background(), listingID)
	require.NoError(t, err)
	assert.Len(t, tripDates, 1)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}
