package booking

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
	byID       map[string]Booking
	byCustomer map[string][]Booking
	nextID     int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]Booking{}, byCustomer: map[string][]Booking{}}
}

func (f *fakeRepository) Create(_ context.Context, b Booking) (Booking, error) {
	f.nextID++
	b.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[b.ID] = b
	f.byCustomer[b.CustomerID] = append(f.byCustomer[b.CustomerID], b)
	return b, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (Booking, error) {
	b, ok := f.byID[id]
	if !ok {
		return Booking{}, ErrNotFound
	}
	return b, nil
}

func (f *fakeRepository) ListByCustomerID(_ context.Context, customerID string) ([]Booking, error) {
	return f.byCustomer[customerID], nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		CustomerID: uuid.NewString(),
		ListingID:  uuid.NewString(),
		TotalPrice: 150.00,
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, 1, created.NumberOfGuests)
	assert.Equal(t, "USD", created.Currency)
	assert.Equal(t, StatusPending, created.Status)
	assert.Equal(t, PaymentStatusPending, created.PaymentStatus)
}

func TestService_Create_KeepsExplicitGuests(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		CustomerID:     uuid.NewString(),
		ListingID:      uuid.NewString(),
		TotalPrice:     150.00,
		NumberOfGuests: 3,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, created.NumberOfGuests)
}

func TestService_ListByCustomerID(t *testing.T) {
	svc := NewService(newFakeRepository())
	customerID := uuid.NewString()

	_, err := svc.Create(context.Background(), CreateInput{CustomerID: customerID, ListingID: uuid.NewString(), TotalPrice: 100})
	require.NoError(t, err)

	bookings, err := svc.ListByCustomerID(context.Background(), customerID)
	require.NoError(t, err)
	assert.Len(t, bookings, 1)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}
