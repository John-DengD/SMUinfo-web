package feedback

import (
	"context"
	"strings"
	"time"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Querier is the subset of the sqlc-generated *gen.Queries the feedback service needs.
type Querier interface {
	InsertFeedback(ctx context.Context, arg gen.InsertFeedbackParams) (gen.Feedback, error)
	ListFeedbackByUser(ctx context.Context, userID *int64) ([]gen.Feedback, error)
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

// CreateReq mirrors FeedbackDTO.CreateReq (camelCase wire contract).
type CreateReq struct {
	Category string  `json:"category"`
	Content  string  `json:"content"`
	Contact  *string `json:"contact"`
}

// Item mirrors FeedbackDTO.Item (camelCase wire contract).
type Item struct {
	ID         int64   `json:"id"`
	UserID     *int64  `json:"userId"`
	Category   string  `json:"category"`
	Content    string  `json:"content"`
	Contact    *string `json:"contact"`
	Status     string  `json:"status"`
	AdminReply *string `json:"adminReply"`
	CreatedAt  *time.Time `json:"createdAt"`
}

func toItem(f gen.Feedback) Item {
	var createdAt *time.Time
	if f.CreatedAt.Valid {
		t := f.CreatedAt.Time
		createdAt = &t
	}
	return Item{
		ID:         f.ID,
		UserID:     f.UserID,
		Category:   f.Category,
		Content:    f.Content,
		Contact:    f.Contact,
		Status:     f.Status,
		AdminReply: f.AdminReply,
		CreatedAt:  createdAt,
	}
}

func (s *Service) Create(ctx context.Context, userID int64, req CreateReq) error {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return httpx.Biz("意见内容不能为空")
	}
	if len([]rune(content)) > 1000 {
		return httpx.Biz("意见内容不能超过 1000 字")
	}

	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		cat = "其他"
	}

	var contact *string
	if req.Contact != nil {
		c := htmlEscape(strings.TrimSpace(*req.Contact))
		if len([]rune(c)) > 64 {
			return httpx.Biz("联系方式不能超过 64 字")
		}
		contact = &c
	}

	uid := userID
	_, err := s.q.InsertFeedback(ctx, gen.InsertFeedbackParams{
		UserID:   &uid,
		Category: cat,
		Content:  htmlEscape(content),
		Contact:  contact,
		Status:   "PENDING",
	})
	return err
}

func (s *Service) ListMine(ctx context.Context, userID int64) ([]Item, error) {
	rows, err := s.q.ListFeedbackByUser(ctx, &userID)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, toItem(r))
	}
	return items, nil
}

// htmlEscape replaces &, <, > with HTML entities (mirrors Spring's HtmlUtils.htmlEscape).
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
