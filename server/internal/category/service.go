package category

import (
	"context"
	"time"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
)

// Querier is the subset of the sqlc-generated *gen.Queries the category service needs.
type Querier interface {
	ListActiveCategories(ctx context.Context) ([]gen.Category, error)
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

// Item mirrors the Category entity wire contract (camelCase JSON).
type Item struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Icon      *string    `json:"icon"`
	SortOrder int32      `json:"sortOrder"`
	Status    string     `json:"status"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

func toItem(c gen.Category) Item {
	var createdAt, updatedAt *time.Time
	if c.CreatedAt.Valid {
		t := c.CreatedAt.Time
		createdAt = &t
	}
	if c.UpdatedAt.Valid {
		t := c.UpdatedAt.Time
		updatedAt = &t
	}
	return Item{
		ID:        c.ID,
		Name:      c.Name,
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
		Status:    c.Status,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// List returns all active categories ordered by sort_order ascending.
func (s *Service) List(ctx context.Context) ([]Item, error) {
	rows, err := s.q.ListActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, toItem(r))
	}
	return items, nil
}
