package report

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Querier is the subset of the sqlc-generated *gen.Queries the report service needs.
type Querier interface {
	GetProductByID(ctx context.Context, id int64) (gen.Product, error)
	InsertReport(ctx context.Context, arg gen.InsertReportParams) (gen.Report, error)
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

// CreateReq mirrors ReportDTO.CreateReq (camelCase wire contract).
type CreateReq struct {
	ProductID int64  `json:"productId"`
	Reason    string `json:"reason"`
}

func (s *Service) Create(ctx context.Context, reporterID int64, req CreateReq) error {
	if req.ProductID == 0 {
		return httpx.Biz("请选择举报的商品")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return httpx.Biz("请填写举报理由")
	}

	_, err := s.q.GetProductByID(ctx, req.ProductID)
	if err != nil {
		if isNoRows(err) {
			return httpx.Biz("商品不存在")
		}
		return err
	}

	_, err = s.q.InsertReport(ctx, gen.InsertReportParams{
		ReporterID: reporterID,
		ProductID:  req.ProductID,
		Reason:     htmlEscape(reason),
		Status:     "PENDING",
	})
	return err
}

func isNoRows(err error) bool {
	return err != nil && (err == pgx.ErrNoRows || strings.Contains(err.Error(), "no rows"))
}

// htmlEscape replaces &, <, > with HTML entities (mirrors Spring's HtmlUtils.htmlEscape).
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
