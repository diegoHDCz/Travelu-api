package category

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository used by service unit tests.
type fakeRepository struct {
	byID   map[string]Category
	bySlug map[string]Category
	byName map[string]Category
	nextID int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byID:   map[string]Category{},
		bySlug: map[string]Category{},
		byName: map[string]Category{},
	}
}

func (f *fakeRepository) Create(_ context.Context, c Category) (Category, error) {
	if _, exists := f.byName[c.Name]; exists {
		return Category{}, ErrNameTaken
	}
	if _, exists := f.bySlug[c.Slug]; exists {
		return Category{}, ErrSlugTaken
	}

	f.nextID++
	c.ID = strconv.FormatInt(f.nextID, 10)
	f.byID[c.ID] = c
	f.byName[c.Name] = c
	f.bySlug[c.Slug] = c
	return c, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id string) (Category, error) {
	c, ok := f.byID[id]
	if !ok {
		return Category{}, ErrNotFound
	}
	return c, nil
}

func (f *fakeRepository) GetBySlug(_ context.Context, slug string) (Category, error) {
	c, ok := f.bySlug[slug]
	if !ok {
		return Category{}, ErrNotFound
	}
	return c, nil
}

func (f *fakeRepository) List(_ context.Context) ([]Category, error) {
	categories := make([]Category, 0, len(f.byID))
	for _, c := range f.byID {
		categories = append(categories, c)
	}
	return categories, nil
}

func TestService_Create(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Create(context.Background(), CreateInput{Name: "Hotels", Slug: "Hotels"})

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "hotels", created.Slug)
	assert.True(t, created.IsActive)
}

func TestService_Create_NameTaken(t *testing.T) {
	svc := NewService(newFakeRepository())
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateInput{Name: "Hotels", Slug: "hotels"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, CreateInput{Name: "Hotels", Slug: "other"})
	assert.ErrorIs(t, err, ErrNameTaken)
}

func TestService_GetBySlug_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository())

	_, err := svc.GetBySlug(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}
