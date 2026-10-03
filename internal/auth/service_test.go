package auth

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/diegoczajka/travelu-api/internal/user"
)

const testJWTSecret = "a-very-long-secret-used-only-for-tests!"

// fakeUserService is an in-memory implementation of UserService.
type fakeUserService struct {
	byEmail map[string]user.User
	byID    map[string]user.User
	nextID  int64
}

func newFakeUserService() *fakeUserService {
	return &fakeUserService{byEmail: map[string]user.User{}, byID: map[string]user.User{}}
}

func (f *fakeUserService) Create(_ context.Context, input user.CreateInput) (user.User, error) {
	if _, exists := f.byEmail[input.Email]; exists {
		return user.User{}, user.ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.MinCost)
	if err != nil {
		return user.User{}, err
	}

	f.nextID++
	now := time.Now().UTC()
	u := user.User{
		ID:           strconv.FormatInt(f.nextID, 10),
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.byEmail[u.Email] = u
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUserService) GetByLogin(_ context.Context, login string) (user.User, error) {
	u, ok := f.byEmail[login]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserService) GetByID(_ context.Context, id string) (user.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

// fakeRefreshTokenRepository is an in-memory implementation of
// RefreshTokenRepository.
type fakeRefreshTokenRepository struct {
	byHash map[string]RefreshToken
	nextID int64
}

func newFakeRefreshTokenRepository() *fakeRefreshTokenRepository {
	return &fakeRefreshTokenRepository{byHash: map[string]RefreshToken{}}
}

func (f *fakeRefreshTokenRepository) Create(_ context.Context, rt RefreshToken) (RefreshToken, error) {
	f.nextID++
	rt.ID = strconv.FormatInt(f.nextID, 10)
	f.byHash[rt.TokenHash] = rt
	return rt, nil
}

func (f *fakeRefreshTokenRepository) GetByTokenHash(_ context.Context, tokenHash string) (RefreshToken, error) {
	rt, ok := f.byHash[tokenHash]
	if !ok {
		return RefreshToken{}, errRefreshTokenNotFound
	}
	return rt, nil
}

func (f *fakeRefreshTokenRepository) Revoke(_ context.Context, id string) error {
	for hash, rt := range f.byHash {
		if rt.ID == id {
			now := time.Now().UTC()
			rt.RevokedAt = &now
			f.byHash[hash] = rt
		}
	}
	return nil
}

func (f *fakeRefreshTokenRepository) RevokeAllForUser(_ context.Context, userID string) error {
	now := time.Now().UTC()
	for hash, rt := range f.byHash {
		if rt.UserID == userID && rt.RevokedAt == nil {
			rt.RevokedAt = &now
			f.byHash[hash] = rt
		}
	}
	return nil
}

func newTestService() (*Service, *fakeUserService, *fakeRefreshTokenRepository) {
	users := newFakeUserService()
	tokens := newFakeRefreshTokenRepository()
	tm := NewTokenManager(testJWTSecret, 15*time.Minute)
	svc := NewService(users, tokens, tm, 30*24*time.Hour)
	return svc, users, tokens
}

func TestService_Login_Success(t *testing.T) {
	svc, users, _ := newTestService()
	ctx := context.Background()
	_, err := users.Create(ctx, user.CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"})
	require.NoError(t, err)

	pair, err := svc.Login(ctx, "diego@example.com", "supersecret", "")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.EqualValues(t, 15*60, pair.ExpiresIn)
}

func TestService_Login_EmbedsDeviceTokenInAccessToken(t *testing.T) {
	svc, users, _ := newTestService()
	ctx := context.Background()
	_, err := users.Create(ctx, user.CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"})
	require.NoError(t, err)

	pair, err := svc.Login(ctx, "diego@example.com", "supersecret", "device-abc")
	require.NoError(t, err)

	_, deviceToken, err := svc.tokens.Parse(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "device-abc", deviceToken)
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, users, _ := newTestService()
	ctx := context.Background()
	_, err := users.Create(ctx, user.CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"})
	require.NoError(t, err)

	_, err = svc.Login(ctx, "diego@example.com", "wrong-password", "")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_Login_UnknownEmail(t *testing.T) {
	svc, _, _ := newTestService()

	_, err := svc.Login(context.Background(), "missing@example.com", "whatever", "")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_Refresh_RotatesAndDetectsReuse(t *testing.T) {
	svc, users, _ := newTestService()
	ctx := context.Background()
	_, err := users.Create(ctx, user.CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"})
	require.NoError(t, err)

	pair, err := svc.Login(ctx, "diego@example.com", "supersecret", "")
	require.NoError(t, err)

	rotated, err := svc.Refresh(ctx, pair.RefreshToken, "")
	require.NoError(t, err)
	assert.NotEqual(t, pair.RefreshToken, rotated.RefreshToken)

	// Reusing the old, already-rotated token is treated as theft: it fails
	// and revokes every refresh token belonging to the user...
	_, err = svc.Refresh(ctx, pair.RefreshToken, "")
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)

	// ...including the one obtained from the legitimate rotation above.
	_, err = svc.Refresh(ctx, rotated.RefreshToken, "")
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func TestService_Refresh_EmbedsDeviceTokenInRotatedAccessToken(t *testing.T) {
	svc, users, _ := newTestService()
	ctx := context.Background()
	_, err := users.Create(ctx, user.CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"})
	require.NoError(t, err)

	pair, err := svc.Login(ctx, "diego@example.com", "supersecret", "device-abc")
	require.NoError(t, err)

	rotated, err := svc.Refresh(ctx, pair.RefreshToken, "device-abc")
	require.NoError(t, err)

	_, deviceToken, err := svc.tokens.Parse(rotated.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "device-abc", deviceToken)
}

func TestService_Logout_RevokesToken(t *testing.T) {
	svc, users, _ := newTestService()
	ctx := context.Background()
	_, err := users.Create(ctx, user.CreateInput{Name: "Diego", Email: "diego@example.com", Password: "supersecret"})
	require.NoError(t, err)

	pair, err := svc.Login(ctx, "diego@example.com", "supersecret", "")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, pair.RefreshToken))

	_, err = svc.Refresh(ctx, pair.RefreshToken, "")
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)

	// Logout is idempotent.
	require.NoError(t, svc.Logout(ctx, pair.RefreshToken))
}
