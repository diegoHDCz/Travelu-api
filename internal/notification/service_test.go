package notification

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
	byID   map[string]Notification
	byUser map[string][]Notification
	nextID int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]Notification{}, byUser: map[string][]Notification{}}
}

func (f *fakeRepository) Create(_ context.Context, n Notification) (Notification, error) {
	f.nextID++
	n.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[n.ID] = n
	f.byUser[n.UserID] = append(f.byUser[n.UserID], n)
	return n, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (Notification, error) {
	n, ok := f.byID[id]
	if !ok {
		return Notification{}, ErrNotFound
	}
	return n, nil
}

func (f *fakeRepository) ListByUserID(_ context.Context, userID string) ([]Notification, error) {
	return f.byUser[userID], nil
}

func (f *fakeRepository) MarkRead(_ context.Context, id string) error {
	n, ok := f.byID[id]
	if !ok {
		return ErrNotFound
	}
	n.IsRead = true
	f.byID[id] = n
	return nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		UserID: uuid.NewString(),
		Title:  "Booking confirmed",
		Body:   "Your booking has been confirmed",
		Type:   TypeBookingConfirmed,
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.False(t, created.IsRead)
}

func TestService_MarkRead(t *testing.T) {
	svc := NewService(newFakeRepository())
	userID := uuid.NewString()

	created, err := svc.Create(context.Background(), CreateInput{UserID: userID, Title: "t", Body: "b", Type: TypeTripReminder})
	require.NoError(t, err)

	require.NoError(t, svc.MarkRead(context.Background(), created.ID))

	notifications, err := svc.ListByUserID(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}
