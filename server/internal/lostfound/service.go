package lostfound

import (
	"context"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// valid type values (mirrors Java TYPES set)
var validTypes = map[string]struct{}{
	"LOST":  {},
	"FOUND": {},
}

// TxBeginner begins a database transaction. Satisfied by *pgxpool.Pool.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Querier is the subset of sqlc-generated *gen.Queries the lostfound service needs.
type Querier interface {
	GetLostFound(ctx context.Context, id int64) (gen.LostFound, error)
	InsertLostFound(ctx context.Context, arg gen.InsertLostFoundParams) (gen.LostFound, error)
	IncrementLostFoundView(ctx context.Context, id int64) error
	SetLostFoundStatus(ctx context.Context, arg gen.SetLostFoundStatusParams) error
	CountLostFound(ctx context.Context, arg gen.CountLostFoundParams) (int64, error)
	ListLostFound(ctx context.Context, arg gen.ListLostFoundParams) ([]gen.LostFound, error)
	ListLostFoundImages(ctx context.Context, lostFoundIds []int64) ([]gen.LostFoundImage, error)
	InsertLostFoundImage(ctx context.Context, arg gen.InsertLostFoundImageParams) error
	DeleteLostFoundImages(ctx context.Context, lostFoundID int64) error
	ListUsersByIDs(ctx context.Context, ids []int64) ([]gen.ListUsersByIDsRow, error)
}

// txWriter is the write operations run inside a transaction.
type txWriter interface {
	InsertLostFound(ctx context.Context, arg gen.InsertLostFoundParams) (gen.LostFound, error)
	InsertLostFoundImage(ctx context.Context, arg gen.InsertLostFoundImageParams) error
}

// Service implements the lost-and-found business logic.
type Service struct {
	q           Querier
	pool        TxBeginner
	newTxWriter func(tx pgx.Tx) txWriter
}

// NewService constructs a Service. pool is used only for the transactional
// Create path; read-only paths go through q.
func NewService(q Querier, pool TxBeginner) *Service {
	return &Service{
		q:           q,
		pool:        pool,
		newTxWriter: func(tx pgx.Tx) txWriter { return gen.New(tx) },
	}
}

// Item mirrors LostFoundDTO.Item (camelCase wire contract).
type Item struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"userId"`
	UserName    *string    `json:"userName"`
	UserCampus  *string    `json:"userCampus"`
	Type        string     `json:"type"`
	TypeText    string     `json:"typeText"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Location    *string    `json:"location"`
	Contact     *string    `json:"contact"`
	Status      string     `json:"status"`
	ViewCount   int32      `json:"viewCount"`
	EventTime   *time.Time `json:"eventTime"`
	CreatedAt   *time.Time `json:"createdAt"`
	Images      []string   `json:"images"`
	Cover       *string    `json:"cover"`
}

// CreateReq mirrors LostFoundDTO.CreateReq.
type CreateReq struct {
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Location    *string    `json:"location"`
	Contact     *string    `json:"contact"`
	EventTime   *time.Time `json:"eventTime"`
	Images      []string   `json:"images"`
}

// ListQuery mirrors LostFoundDTO.ListQuery.
type ListQuery struct {
	Type    *string
	Keyword *string
	Page    *int32
	Size    *int32
}

// List replicates LostFoundService.list: always filters status=OPEN, optional
// type + keyword, paginated, ordered by created_at DESC.
func (s *Service) List(ctx context.Context, q ListQuery) (httpx.Page, error) {
	page := int32(1)
	if q.Page != nil && *q.Page >= 1 {
		page = *q.Page
	}
	size := int32(12)
	if q.Size != nil && *q.Size >= 1 {
		size = *q.Size
	}
	if size > 100 {
		size = 100
	}
	off64 := (int64(page) - 1) * int64(size)
	if off64 > int64(^uint32(0)>>1) {
		off64 = int64(^uint32(0) >> 1)
	}
	offset := int32(off64)

	typ := blankToNil(q.Type)
	keyword := blankToNil(q.Keyword)

	total, err := s.q.CountLostFound(ctx, gen.CountLostFoundParams{
		Type:    typ,
		Keyword: keyword,
	})
	if err != nil {
		return httpx.Page{}, err
	}

	rows, err := s.q.ListLostFound(ctx, gen.ListLostFoundParams{
		Type:    typ,
		Keyword: keyword,
		Off:     offset,
		Lim:     size,
	})
	if err != nil {
		return httpx.Page{}, err
	}

	items, err := s.enrich(ctx, rows)
	if err != nil {
		return httpx.Page{}, err
	}
	return httpx.Page{Total: total, Records: items}, nil
}

// Detail replicates LostFoundService.detail: must be OPEN, increments view_count,
// returns the enriched item with incremented count.
func (s *Service) Detail(ctx context.Context, id int64) (Item, error) {
	row, err := s.q.GetLostFound(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, httpx.Biz("内容不存在")
		}
		return Item{}, err
	}
	if row.Status != "OPEN" {
		return Item{}, httpx.Biz("内容不存在")
	}
	if err := s.q.IncrementLostFoundView(ctx, id); err != nil {
		return Item{}, err
	}
	row.ViewCount++ // reflect increment in response (matches Java)
	items, err := s.enrich(ctx, []gen.LostFound{row})
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

// Create replicates LostFoundService.create: validates, inserts in a transaction,
// then calls detail (which also increments view_count, matching Java).
func (s *Service) Create(ctx context.Context, userID int64, req CreateReq) (Item, error) {
	typ, err := normalizeType(req.Type)
	if err != nil {
		return Item{}, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return Item{}, httpx.Biz("参数错误")
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		return Item{}, httpx.Biz("参数错误")
	}

	title = html.EscapeString(title)
	description = html.EscapeString(description)

	var location *string
	if req.Location != nil {
		v := html.EscapeString(strings.TrimSpace(*req.Location))
		location = &v
	}
	var contact *string
	if req.Contact != nil {
		v := html.EscapeString(strings.TrimSpace(*req.Contact))
		contact = &v
	}

	var eventTime pgtype.Timestamptz
	if req.EventTime != nil {
		eventTime = pgtype.Timestamptz{Time: *req.EventTime, Valid: true}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Item{}, err
	}
	defer tx.Rollback(ctx) // no-op after Commit
	qtx := s.newTxWriter(tx)

	row, err := qtx.InsertLostFound(ctx, gen.InsertLostFoundParams{
		UserID:      userID,
		Type:        typ,
		Title:       title,
		Description: description,
		Location:    location,
		Contact:     contact,
		Status:      "OPEN",
		ViewCount:   0,
		EventTime:   eventTime,
	})
	if err != nil {
		return Item{}, err
	}

	if err := saveLostFoundImages(ctx, qtx, row.ID, req.Images); err != nil {
		return Item{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Item{}, err
	}

	// Java calls detail(row.getId()) after commit, which increments view_count.
	return s.Detail(ctx, row.ID)
}

// Close replicates LostFoundService.close: sets status to CLOSED. Owner or admin.
func (s *Service) Close(ctx context.Context, id, userID int64, isAdmin bool) error {
	row, err := s.q.GetLostFound(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("内容不存在")
		}
		return err
	}
	if !isAdmin && row.UserID != userID {
		return httpx.NewBiz(403, "无权操作")
	}
	return s.q.SetLostFoundStatus(ctx, gen.SetLostFoundStatusParams{
		ID:     id,
		Status: "CLOSED",
	})
}

// enrich batch-loads images and poster info for a slice of LostFound rows.
func (s *Service) enrich(ctx context.Context, rows []gen.LostFound) ([]Item, error) {
	if len(rows) == 0 {
		return make([]Item, 0), nil
	}

	ids := make([]int64, 0, len(rows))
	userSet := map[int64]struct{}{}
	for _, r := range rows {
		ids = append(ids, r.ID)
		userSet[r.UserID] = struct{}{}
	}

	imgs, err := s.q.ListLostFoundImages(ctx, ids)
	if err != nil {
		return nil, err
	}
	imageMap := map[int64][]string{}
	for _, img := range imgs {
		imageMap[img.LostFoundID] = append(imageMap[img.LostFoundID], img.ImageUrl)
	}

	users, err := s.q.ListUsersByIDs(ctx, keys(userSet))
	if err != nil {
		return nil, err
	}
	userMap := map[int64]gen.ListUsersByIDsRow{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]Item, 0, len(rows))
	for _, r := range rows {
		it := Item{
			ID:          r.ID,
			UserID:      r.UserID,
			Type:        r.Type,
			TypeText:    typeText(r.Type),
			Title:       r.Title,
			Description: r.Description,
			Location:    r.Location,
			Contact:     r.Contact,
			Status:      r.Status,
			ViewCount:   r.ViewCount,
			EventTime:   timePtr(r.EventTime),
			CreatedAt:   timePtr(r.CreatedAt),
		}
		if u, ok := userMap[r.UserID]; ok {
			n := u.Name
			it.UserName = &n
			it.UserCampus = u.Campus
		}
		imgList := imageMap[r.ID]
		it.Images = make([]string, 0, len(imgList))
		it.Images = append(it.Images, imgList...)
		if len(imgList) > 0 {
			cover := imgList[0]
			it.Cover = &cover
		}
		result = append(result, it)
	}
	return result, nil
}

func saveLostFoundImages(ctx context.Context, q txWriter, lostFoundID int64, urls []string) error {
	for i, url := range urls {
		if strings.TrimSpace(url) == "" {
			continue
		}
		if err := q.InsertLostFoundImage(ctx, gen.InsertLostFoundImageParams{
			LostFoundID: lostFoundID,
			ImageUrl:    url,
			SortOrder:   int32(i),
		}); err != nil {
			return err
		}
	}
	return nil
}

func normalizeType(t string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(t))
	if _, ok := validTypes[normalized]; !ok {
		return "", httpx.Biz("类型不正确")
	}
	return normalized, nil
}

func typeText(t string) string {
	if t == "LOST" {
		return "寻物"
	}
	return "招领"
}

func timePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
}

func keys(m map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
