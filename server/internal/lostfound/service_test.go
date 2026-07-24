package lostfound

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// ---- stub Querier ----

type stubQuerier struct {
	rows       map[int64]gen.LostFound
	images     []gen.LostFoundImage
	users      []gen.ListUsersByIDsRow
	countResult int64
	listResult  []gen.LostFound

	lastCountParams gen.CountLostFoundParams
	lastListParams  gen.ListLostFoundParams
	insertedImages  []gen.InsertLostFoundImageParams
	deletedFor      []int64
	viewIncFor      []int64
	statusSet       []gen.SetLostFoundStatusParams
}

func (s *stubQuerier) GetLostFound(_ context.Context, id int64) (gen.LostFound, error) {
	r, ok := s.rows[id]
	if !ok {
		return gen.LostFound{}, pgx.ErrNoRows
	}
	return r, nil
}

func (s *stubQuerier) InsertLostFound(_ context.Context, arg gen.InsertLostFoundParams) (gen.LostFound, error) {
	row := gen.LostFound{
		ID:          999,
		UserID:      arg.UserID,
		Type:        arg.Type,
		Title:       arg.Title,
		Description: arg.Description,
		Location:    arg.Location,
		Contact:     arg.Contact,
		Status:      arg.Status,
		ViewCount:   arg.ViewCount,
		EventTime:   arg.EventTime,
	}
	if s.rows == nil {
		s.rows = map[int64]gen.LostFound{}
	}
	s.rows[999] = row
	return row, nil
}

func (s *stubQuerier) IncrementLostFoundView(_ context.Context, id int64) error {
	s.viewIncFor = append(s.viewIncFor, id)
	return nil
}

func (s *stubQuerier) SetLostFoundStatus(_ context.Context, arg gen.SetLostFoundStatusParams) error {
	s.statusSet = append(s.statusSet, arg)
	return nil
}

func (s *stubQuerier) CountLostFound(_ context.Context, arg gen.CountLostFoundParams) (int64, error) {
	s.lastCountParams = arg
	return s.countResult, nil
}

func (s *stubQuerier) ListLostFound(_ context.Context, arg gen.ListLostFoundParams) ([]gen.LostFound, error) {
	s.lastListParams = arg
	return s.listResult, nil
}

func (s *stubQuerier) ListLostFoundImages(_ context.Context, _ []int64) ([]gen.LostFoundImage, error) {
	return s.images, nil
}

func (s *stubQuerier) InsertLostFoundImage(_ context.Context, arg gen.InsertLostFoundImageParams) error {
	s.insertedImages = append(s.insertedImages, arg)
	return nil
}

func (s *stubQuerier) DeleteLostFoundImages(_ context.Context, id int64) error {
	s.deletedFor = append(s.deletedFor, id)
	return nil
}

func (s *stubQuerier) ListUsersByIDs(_ context.Context, _ []int64) ([]gen.ListUsersByIDsRow, error) {
	return s.users, nil
}

// ---- fake transaction plumbing (mirrors product test pattern) ----

type fakeTx struct {
	pgx.Tx
	committed  bool
	rolledBack bool
}

func (t *fakeTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *fakeTx) Rollback(context.Context) error { t.rolledBack = true; return nil }

type fakeBeginner struct{ tx *fakeTx }

func (b *fakeBeginner) Begin(context.Context) (pgx.Tx, error) {
	if b.tx == nil {
		b.tx = &fakeTx{}
	}
	return b.tx, nil
}

func newTestService(stub *stubQuerier) *Service {
	svc := NewService(stub, &fakeBeginner{})
	svc.newTxWriter = func(pgx.Tx) txWriter { return stub }
	return svc
}

// ---- tests: type validation ----

func TestNormalizeTypeRejectsInvalid(t *testing.T) {
	svc := newTestService(&stubQuerier{})
	_, err := svc.Create(context.Background(), 1, CreateReq{
		Type:        "UNKNOWN",
		Title:       "test",
		Description: "desc",
	})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "类型不正确" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestNormalizeTypeLOSTCaseInsensitive(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	_, err := svc.Create(context.Background(), 1, CreateReq{
		Type:        "lost",
		Title:       "test",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.rows[999].Type != "LOST" {
		t.Fatalf("expected LOST, got %q", stub.rows[999].Type)
	}
}

func TestNormalizeTypeFOUNDAccepted(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	_, err := svc.Create(context.Background(), 1, CreateReq{
		Type:        "FOUND",
		Title:       "test",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.rows[999].Type != "FOUND" {
		t.Fatalf("expected FOUND, got %q", stub.rows[999].Type)
	}
}

// ---- tests: ownership guard ----

func TestCloseByOwnerSucceeds(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 42, Status: "OPEN"},
		},
	}
	svc := newTestService(stub)
	if err := svc.Close(context.Background(), 1, 42, false); err != nil {
		t.Fatalf("owner close failed: %v", err)
	}
	if len(stub.statusSet) != 1 || stub.statusSet[0].Status != "CLOSED" {
		t.Fatal("expected CLOSED status to be set")
	}
}

func TestCloseByAdminSucceeds(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 42, Status: "OPEN"},
		},
	}
	svc := newTestService(stub)
	// different user but isAdmin=true
	if err := svc.Close(context.Background(), 1, 99, true); err != nil {
		t.Fatalf("admin close failed: %v", err)
	}
}

func TestCloseByNonOwnerReturns403(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 42, Status: "OPEN"},
		},
	}
	svc := newTestService(stub)
	err := svc.Close(context.Background(), 1, 99, false)
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Code != 403 {
		t.Fatalf("expected 403, got %d", be.Code)
	}
	if be.Msg != "无权操作" {
		t.Fatalf("expected '无权操作', got %q", be.Msg)
	}
}

func TestCloseNotFoundReturnsError(t *testing.T) {
	svc := newTestService(&stubQuerier{rows: map[int64]gen.LostFound{}})
	err := svc.Close(context.Background(), 999, 1, false)
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "内容不存在" {
		t.Fatalf("expected '内容不存在', got %q", be.Msg)
	}
}

// ---- tests: detail ----

func TestDetailClosedReturnsNotFound(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 1, Status: "CLOSED"},
		},
	}
	svc := newTestService(stub)
	_, err := svc.Detail(context.Background(), 1)
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "内容不存在" {
		t.Fatalf("expected '内容不存在', got %q", be.Msg)
	}
}

func TestDetailIncrementsViewCount(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 1, Status: "OPEN", ViewCount: 5},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ViewCount != 6 {
		t.Fatalf("expected viewCount=6, got %d", item.ViewCount)
	}
	if len(stub.viewIncFor) != 1 || stub.viewIncFor[0] != 1 {
		t.Fatal("expected IncrementLostFoundView to be called with id=1")
	}
}

// ---- tests: list pagination defaults ----

func TestListPaginationDefaults(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	if _, err := svc.List(context.Background(), ListQuery{}); err != nil {
		t.Fatal(err)
	}
	if stub.lastListParams.Lim != 12 {
		t.Fatalf("default size want 12, got %d", stub.lastListParams.Lim)
	}
	if stub.lastListParams.Off != 0 {
		t.Fatalf("default page=1 offset want 0, got %d", stub.lastListParams.Off)
	}
}

func TestListPaginationOffset(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	page, size := int32(3), int32(10)
	if _, err := svc.List(context.Background(), ListQuery{Page: &page, Size: &size}); err != nil {
		t.Fatal(err)
	}
	// offset = (3-1)*10 = 20
	if stub.lastListParams.Off != 20 {
		t.Fatalf("want offset=20, got %d", stub.lastListParams.Off)
	}
}

// ---- tests: image creation is atomic ----

func TestCreateImagesInsertedWithSortOrder(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	now := time.Now()
	_, err := svc.Create(context.Background(), 1, CreateReq{
		Type:        "LOST",
		Title:       "lost phone",
		Description: "dropped somewhere",
		EventTime:   &now,
		Images:      []string{"http://a.com/1.jpg", "http://a.com/2.jpg"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.insertedImages) != 2 {
		t.Fatalf("expected 2 image inserts, got %d", len(stub.insertedImages))
	}
	if stub.insertedImages[0].SortOrder != 0 {
		t.Fatalf("first image sort_order want 0, got %d", stub.insertedImages[0].SortOrder)
	}
	if stub.insertedImages[1].SortOrder != 1 {
		t.Fatalf("second image sort_order want 1, got %d", stub.insertedImages[1].SortOrder)
	}
}

// ---- tests: mapping ----

func TestEnrichMapsTypeText(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 1, Status: "OPEN", Type: "LOST", ViewCount: 0},
			2: {ID: 2, UserID: 1, Status: "OPEN", Type: "FOUND", ViewCount: 0},
		},
	}
	svc := newTestService(stub)
	item1, err := svc.Detail(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if item1.TypeText != "寻物" {
		t.Fatalf("LOST typeText want '寻物', got %q", item1.TypeText)
	}
	item2, err := svc.Detail(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if item2.TypeText != "招领" {
		t.Fatalf("FOUND typeText want '招领', got %q", item2.TypeText)
	}
}

func TestEnrichEmptyImagesIsSlice(t *testing.T) {
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {ID: 1, UserID: 1, Status: "OPEN", Type: "LOST"},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if item.Images == nil {
		t.Fatal("images should be non-nil empty slice, not nil")
	}
	if len(item.Images) != 0 {
		t.Fatalf("expected 0 images, got %d", len(item.Images))
	}
	if item.Cover != nil {
		t.Fatal("cover should be nil when no images")
	}
}

func TestEnrichEventTimeNullableRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	stub := &stubQuerier{
		rows: map[int64]gen.LostFound{
			1: {
				ID:        1,
				UserID:    1,
				Status:    "OPEN",
				Type:      "LOST",
				EventTime: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if item.EventTime == nil {
		t.Fatal("eventTime should not be nil")
	}
	if !item.EventTime.Equal(now) {
		t.Fatalf("eventTime roundtrip mismatch: want %v got %v", now, *item.EventTime)
	}
}

func TestHtmlEscapingOnCreate(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	_, err := svc.Create(context.Background(), 1, CreateReq{
		Type:        "LOST",
		Title:       "<script>alert(1)</script>",
		Description: "<b>test</b>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row := stub.rows[999]
	if row.Title != "&lt;script&gt;alert(1)&lt;/script&gt;" {
		t.Fatalf("title not escaped: %q", row.Title)
	}
	if row.Description != "&lt;b&gt;test&lt;/b&gt;" {
		t.Fatalf("description not escaped: %q", row.Description)
	}
}
