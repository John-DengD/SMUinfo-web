package announcement

import (
	"context"
	"testing"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type stubQuerier struct {
	row *gen.Announcement
}

func (s *stubQuerier) GetActiveAnnouncement(_ context.Context) (gen.Announcement, error) {
	if s.row == nil {
		return gen.Announcement{}, pgx.ErrNoRows
	}
	return *s.row, nil
}

func TestActiveReturnsNilWhenNoneExists(t *testing.T) {
	svc := NewService(&stubQuerier{row: nil})
	item, err := svc.Active(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != nil {
		t.Fatal("expected nil item when no active announcement")
	}
}

func TestActiveReturnsItemWhenExists(t *testing.T) {
	createdBy := int64(1)
	row := &gen.Announcement{
		ID:        42,
		Title:     "Test",
		Content:   "Hello",
		Status:    "ACTIVE",
		CreatedBy: &createdBy,
		CreatedAt: pgtype.Timestamptz{},
		UpdatedAt: pgtype.Timestamptz{},
	}
	svc := NewService(&stubQuerier{row: row})
	item, err := svc.Active(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.ID != 42 {
		t.Fatalf("expected ID 42, got %d", item.ID)
	}
	if item.Title != "Test" {
		t.Fatalf("expected title 'Test', got %q", item.Title)
	}
}
