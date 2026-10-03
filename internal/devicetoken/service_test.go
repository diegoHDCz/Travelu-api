package devicetoken

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
	byID    map[string]DeviceToken
	byUser  map[string][]DeviceToken
	byToken map[string]bool
	nextID  int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]DeviceToken{}, byUser: map[string][]DeviceToken{}, byToken: map[string]bool{}}
}

func (f *fakeRepository) Create(_ context.Context, dt DeviceToken) (DeviceToken, error) {
	if f.byToken[dt.Token] {
		return DeviceToken{}, ErrTokenTaken
	}

	f.nextID++
	dt.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[dt.ID] = dt
	f.byUser[dt.UserID] = append(f.byUser[dt.UserID], dt)
	f.byToken[dt.Token] = true
	return dt, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (DeviceToken, error) {
	dt, ok := f.byID[id]
	if !ok {
		return DeviceToken{}, ErrNotFound
	}
	return dt, nil
}

func (f *fakeRepository) ListByUserID(_ context.Context, userID string) ([]DeviceToken, error) {
	return f.byUser[userID], nil
}

func (f *fakeRepository) Deactivate(_ context.Context, id string) error {
	dt, ok := f.byID[id]
	if !ok {
		return ErrNotFound
	}
	dt.IsActive = false
	f.byID[id] = dt
	return nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		UserID:   uuid.NewString(),
		Token:    "abc123",
		Platform: PlatformAndroid,
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.True(t, created.IsActive)
}

func TestService_Create_TokenTaken(t *testing.T) {
	svc := NewService(newFakeRepository())
	ctx := context.Background()
	input := CreateInput{UserID: uuid.NewString(), Token: "abc123", Platform: PlatformAndroid}

	_, err := svc.Create(ctx, input)
	require.NoError(t, err)

	_, err = svc.Create(ctx, input)
	assert.ErrorIs(t, err, ErrTokenTaken)
}

func TestService_Deactivate(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{UserID: uuid.NewString(), Token: "abc123", Platform: PlatformIOS})
	require.NoError(t, err)

	require.NoError(t, svc.Deactivate(context.Background(), created.ID))
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}
