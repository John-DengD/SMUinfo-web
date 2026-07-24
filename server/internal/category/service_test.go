package category

import (
	"context"
	"testing"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

// stubQuerier implements Querier for unit tests.
type stubQuerier struct {
	rows []gen.Category
}

func (s *stubQuerier) ListActiveCategories(_ context.Context) ([]gen.Category, error) {
	return s.rows, nil
}

func TestListReturnsItemsInOrder(t *testing.T) {
	icon := "icon-book"
	stub := &stubQuerier{
		rows: []gen.Category{
			{ID: 1, Name: "图书", Icon: &icon, SortOrder: 1, Status: "ACTIVE", CreatedAt: pgtype.Timestamptz{}, UpdatedAt: pgtype.Timestamptz{}},
			{ID: 2, Name: "电子", Icon: nil, SortOrder: 2, Status: "ACTIVE", CreatedAt: pgtype.Timestamptz{}, UpdatedAt: pgtype.Timestamptz{}},
		},
	}
	svc := NewService(stub)
	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "图书" {
		t.Fatalf("expected first item '图书', got %q", items[0].Name)
	}
	if items[1].Icon != nil {
		t.Fatal("expected nil icon for second item")
	}
}

func TestListEmptyReturnsEmptySlice(t *testing.T) {
	stub := &stubQuerier{rows: []gen.Category{}}
	svc := NewService(stub)
	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}
