package feedback

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
	"github.com/jackc/pgx/v5/pgtype"
)

type stubQuerier struct {
	inserted gen.InsertFeedbackParams
	rows     []gen.Feedback
}

func (s *stubQuerier) InsertFeedback(_ context.Context, arg gen.InsertFeedbackParams) (gen.Feedback, error) {
	s.inserted = arg
	return gen.Feedback{
		ID:       1,
		UserID:   arg.UserID,
		Category: arg.Category,
		Content:  arg.Content,
		Contact:  arg.Contact,
		Status:   arg.Status,
	}, nil
}

func (s *stubQuerier) ListFeedbackByUser(_ context.Context, userID *int64) ([]gen.Feedback, error) {
	return s.rows, nil
}

func TestCreateEmptyContentReturnsError(t *testing.T) {
	svc := NewService(&stubQuerier{})
	err := svc.Create(context.Background(), 1, CreateReq{Content: "   "})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "意见内容不能为空" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestCreateContentTooLongReturnsError(t *testing.T) {
	svc := NewService(&stubQuerier{})
	// 1001 runes
	longContent := strings.Repeat("x", 1001)
	err := svc.Create(context.Background(), 1, CreateReq{Content: longContent})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "意见内容不能超过 1000 字" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestCreateDefaultCategory(t *testing.T) {
	stub := &stubQuerier{}
	svc := NewService(stub)
	err := svc.Create(context.Background(), 1, CreateReq{Content: "some feedback"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.inserted.Category != "其他" {
		t.Fatalf("expected default category '其他', got %q", stub.inserted.Category)
	}
}

func TestCreateHtmlEscapesContent(t *testing.T) {
	stub := &stubQuerier{}
	svc := NewService(stub)
	err := svc.Create(context.Background(), 1, CreateReq{Content: "<b>test</b>"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.inserted.Content != "&lt;b&gt;test&lt;/b&gt;" {
		t.Fatalf("unexpected escaped content: %q", stub.inserted.Content)
	}
}

func TestListMineReturnsItems(t *testing.T) {
	uid := int64(1)
	stub := &stubQuerier{
		rows: []gen.Feedback{
			{ID: 1, UserID: &uid, Category: "Bug", Content: "test", Status: "PENDING", CreatedAt: pgtype.Timestamptz{}},
		},
	}
	svc := NewService(stub)
	items, err := svc.ListMine(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Category != "Bug" {
		t.Fatalf("unexpected category: %q", items[0].Category)
	}
}
