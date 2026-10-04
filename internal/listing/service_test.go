package listing

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
	byID   map[string]Listing
	nextID int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]Listing{}}
}

func (f *fakeRepository) Create(_ context.Context, l Listing) (Listing, error) {
	f.nextID++
	l.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[l.ID] = l
	return l, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (Listing, error) {
	l, ok := f.byID[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	return l, nil
}

func (f *fakeRepository) ListActive(_ context.Context) ([]Listing, error) {
	var active []Listing
	for _, l := range f.byID {
		if l.IsActive {
			active = append(active, l)
		}
	}
	return active, nil
}

func (f *fakeRepository) Update(_ context.Context, l Listing) (Listing, error) {
	if _, ok := f.byID[l.ID]; !ok {
		return Listing{}, ErrNotFound
	}
	f.byID[l.ID] = l
	return l, nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		VendorID:    uuid.NewString(),
		Title:       "Beach Hotel",
		Description: "A nice hotel",
		Category:    CategoryHotel,
		Location:    "Rio de Janeiro",
		Price:       199.90,
	})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "USD", created.Currency)
	assert.True(t, created.IsActive)
}

func TestService_Create_KeepsExplicitCurrency(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		VendorID:    uuid.NewString(),
		Title:       "Beach Hotel",
		Description: "A nice hotel",
		Category:    CategoryHotel,
		Location:    "Rio de Janeiro",
		Price:       199.90,
		Currency:    "brl",
	})

	require.NoError(t, err)
	assert.Equal(t, "BRL", created.Currency)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_Update(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{
		VendorID:    uuid.NewString(),
		Title:       "Beach Hotel",
		Description: "A nice hotel",
		Category:    CategoryHotel,
		Location:    "Rio de Janeiro",
		Price:       199.90,
	})
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), UpdateInput{
		ID:          created.ID,
		Title:       "Beach Resort",
		Description: "An even nicer hotel",
		Category:    CategoryHotel,
		Location:    "Rio de Janeiro",
		Price:       249.90,
		IsActive:    true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Beach Resort", updated.Title)
	assert.Equal(t, 249.90, updated.Price)
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.Update(context.Background(), UpdateInput{
		ID:       "missing",
		Title:    "Beach Resort",
		Category: CategoryHotel,
		Location: "Rio de Janeiro",
		Price:    249.90,
	})
	assert.ErrorIs(t, err, ErrNotFound)
}
