package user

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository used by service unit tests.
type fakeRepository struct {
	byID       map[string]User
	byEmail    map[string]User
	byPhone    map[string]User
	byUsername map[string]User
	nextID     int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byID:       map[string]User{},
		byEmail:    map[string]User{},
		byPhone:    map[string]User{},
		byUsername: map[string]User{},
	}
}

func (f *fakeRepository) Create(_ context.Context, u User) (User, error) {
	if u.Email != "" {
		if _, exists := f.byEmail[u.Email]; exists {
			return User{}, ErrEmailTaken
		}
	}
	if u.Phone != "" {
		if _, exists := f.byPhone[u.Phone]; exists {
			return User{}, ErrPhoneTaken
		}
	}
	if u.Username != "" {
		if _, exists := f.byUsername[u.Username]; exists {
			return User{}, ErrUsernameTaken
		}
	}

	f.nextID++
	u.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[u.ID] = u
	if u.Email != "" {
		f.byEmail[u.Email] = u
	}
	if u.Phone != "" {
		f.byPhone[u.Phone] = u
	}
	if u.Username != "" {
		f.byUsername[u.Username] = u
	}
	return u, nil
}

func (f *fakeRepository) GetByEmail(_ context.Context, email string) (User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeRepository) GetByPhone(_ context.Context, phone string) (User, error) {
	u, ok := f.byPhone[phone]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeRepository) GetByUsername(_ context.Context, username string) (User, error) {
	u, ok := f.byUsername[username]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (User, error) {
	u, ok := f.byID[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		Name:     "Diego",
		Email:    "diego@example.com",
		Password: "supersecret",
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "diego@example.com", created.Email)
	assert.NotEqual(t, "supersecret", created.PasswordHash)
}

func TestService_Create_EmailTaken(t *testing.T) {
	svc := NewService(newFakeRepository())
	ctx := context.Background()
	input := CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"}

	_, err := svc.Create(ctx, input)
	require.NoError(t, err)

	_, err = svc.Create(ctx, input)
	assert.ErrorIs(t, err, ErrEmailTaken)
}

func TestService_Create_WithPhoneOnly_SanitizesAndStores(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		Name:     "Diego",
		Phone:    "+55 (11) 99999-8888",
		Password: "supersecret",
	})

	require.NoError(t, err)
	assert.Equal(t, "+5511999998888", created.Phone)
	assert.Empty(t, created.Email)
}

func TestService_Create_InvalidPhone(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.Create(context.Background(), CreateInput{
		Name:     "Diego",
		Phone:    "not-a-phone",
		Password: "supersecret",
	})

	assert.ErrorIs(t, err, ErrInvalidPhone)
}

func TestService_Create_WithUsernameOnly(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		Name:     "Diego",
		Username: "DiegoC",
		Password: "supersecret",
	})

	require.NoError(t, err)
	assert.Equal(t, "diegoc", created.Username)
}

func TestService_GetByEmail_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByEmail(context.Background(), "missing@example.com")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_GetByLogin(t *testing.T) {
	svc := NewService(newFakeRepository())
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateInput{Name: "Diego", Email: "diego@example.com", Phone: "+5511999998888", Username: "diegoc", Password: "supersecret"})
	require.NoError(t, err)

	byEmail, err := svc.GetByLogin(ctx, "diego@example.com")
	require.NoError(t, err)
	assert.Equal(t, "diegoc", byEmail.Username)

	byPhone, err := svc.GetByLogin(ctx, "+55 11 99999-8888")
	require.NoError(t, err)
	assert.Equal(t, "diegoc", byPhone.Username)

	byUsername, err := svc.GetByLogin(ctx, "diegoc")
	require.NoError(t, err)
	assert.Equal(t, "diego@example.com", byUsername.Email)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "999")
	assert.ErrorIs(t, err, ErrNotFound)
}
