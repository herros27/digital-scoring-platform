package service

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
)

type mockCategoryRepo struct {
	categories map[string]*domain.Category
}

func (m *mockCategoryRepo) Create(c *domain.Category) error {
	m.categories[c.ID] = c
	return nil
}

func (m *mockCategoryRepo) GetByID(id string) (*domain.Category, error) {
	if cat, ok := m.categories[id]; ok {
		return cat, nil
	}
	return nil, nil
}

func (m *mockCategoryRepo) ListByCompetition(compID string) ([]domain.Category, error) {
	var list []domain.Category
	for _, c := range m.categories {
		if c.CompetitionID == compID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func TestCategoryService_ValidationAndCRUD(t *testing.T) {
	repo := &mockCategoryRepo{categories: make(map[string]*domain.Category)}
	svc := NewCategoryService(repo)

	// 1. Validation errors
	if err := svc.CreateCategory(&domain.Category{}); err == nil {
		t.Errorf("expected error for empty competition_id")
	}
	if err := svc.CreateCategory(&domain.Category{CompetitionID: "c1"}); err == nil {
		t.Errorf("expected error for empty name")
	}
	if err := svc.CreateCategory(&domain.Category{CompetitionID: "c1", Name: "Tanding A"}); err == nil {
		t.Errorf("expected error for empty gender")
	}

	// 2. Successful creation
	c1 := &domain.Category{
		ID:            "cat-1",
		CompetitionID: "c1",
		Name:          "Tanding A",
		Gender:        "male",
		WeightClass:   "A",
	}
	if err := svc.CreateCategory(c1); err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// 3. GetByID
	cat, err := svc.GetCategoryByID("cat-1")
	if err != nil {
		t.Fatalf("GetCategoryByID failed: %v", err)
	}
	if cat.Name != "Tanding A" {
		t.Errorf("expected name 'Tanding A', got %s", cat.Name)
	}

	// 4. GetByID not found
	_, errNotFound := svc.GetCategoryByID("cat-unknown")
	if errNotFound != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got: %v", errNotFound)
	}

	// 5. ListCategories
	list, err := svc.ListCategories("c1")
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 category, got %d", len(list))
	}
}
