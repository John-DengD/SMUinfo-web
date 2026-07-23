package report

import (
	"context"
	"errors"
	"testing"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
	"github.com/jackc/pgx/v5"
)

type stubQuerier struct {
	product *gen.Product
}

func (s *stubQuerier) GetProductByID(_ context.Context, id int64) (gen.Product, error) {
	if s.product == nil {
		return gen.Product{}, pgx.ErrNoRows
	}
	return *s.product, nil
}

func (s *stubQuerier) InsertReport(_ context.Context, arg gen.InsertReportParams) (gen.Report, error) {
	return gen.Report{
		ID:         1,
		ReporterID: arg.ReporterID,
		ProductID:  arg.ProductID,
		Reason:     arg.Reason,
		Status:     arg.Status,
	}, nil
}

func TestCreateReportProductNotFound(t *testing.T) {
	svc := NewService(&stubQuerier{product: nil})
	err := svc.Create(context.Background(), 1, CreateReq{ProductID: 99, Reason: "scam"})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "商品不存在" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestCreateReportEmptyReason(t *testing.T) {
	p := gen.Product{ID: 1}
	svc := NewService(&stubQuerier{product: &p})
	err := svc.Create(context.Background(), 1, CreateReq{ProductID: 1, Reason: "   "})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "请填写举报理由" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestCreateReportHtmlEscapesReason(t *testing.T) {
	p := gen.Product{ID: 1}
	svc := NewService(&stubQuerier{product: &p})
	err := svc.Create(context.Background(), 1, CreateReq{ProductID: 1, Reason: "<script>"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHtmlEscape(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<b>", "&lt;b&gt;"},
		{"a&b", "a&amp;b"},
		{"normal", "normal"},
	}
	for _, tc := range cases {
		got := htmlEscape(tc.in)
		if got != tc.want {
			t.Errorf("htmlEscape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
