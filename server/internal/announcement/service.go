package announcement

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
)

// Querier is the subset of the sqlc-generated *gen.Queries the announcement service needs.
type Querier interface {
	GetActiveAnnouncement(ctx context.Context) (gen.Announcement, error)
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

// Item mirrors AnnouncementDTO.Item (camelCase wire contract).
type Item struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Status    string     `json:"status"`
	CreatedBy *int64     `json:"createdBy"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

func toItem(a gen.Announcement) Item {
	var createdAt, updatedAt *time.Time
	if a.CreatedAt.Valid {
		t := a.CreatedAt.Time
		createdAt = &t
	}
	if a.UpdatedAt.Valid {
		t := a.UpdatedAt.Time
		updatedAt = &t
	}
	return Item{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		Status:    a.Status,
		CreatedBy: a.CreatedBy,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// Active returns the most recently created active announcement, or nil if none.
func (s *Service) Active(ctx context.Context) (*Item, error) {
	row, err := s.q.GetActiveAnnouncement(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item := toItem(row)
	return &item, nil
}
