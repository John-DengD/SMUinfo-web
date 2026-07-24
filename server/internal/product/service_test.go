package product

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// stubQuerier records the params passed to the data layer and returns canned rows.
type stubQuerier struct {
	products map[int64]gen.Product
	images   []gen.ProductImage
	users    []gen.ListUsersByIDsRow
	cats     []gen.ListCategoriesByIDsRow
	favIDs   []int64

	listParams  gen.ListProductsParams
	countParams gen.CountProductsParams
	countResult int64
	listResult  []gen.Product

	insertImages  []gen.InsertProductImageParams
	deletedImgFor []int64
	viewIncFor    []int64
	statusSet     []gen.SetProductStatusParams
	insertedComm  gen.InsertProductCommentParams
	comments      []gen.ProductComment
}

func (s *stubQuerier) GetProduct(_ context.Context, id int64) (gen.Product, error) {
	p, ok := s.products[id]
	if !ok {
		return gen.Product{}, pgx.ErrNoRows
	}
	return p, nil
}
func (s *stubQuerier) InsertProduct(_ context.Context, arg gen.InsertProductParams) (gen.Product, error) {
	p := gen.Product{ID: 100, SellerID: arg.SellerID, CategoryID: arg.CategoryID, Title: arg.Title,
		Description: arg.Description, Price: arg.Price, OriginalPrice: arg.OriginalPrice,
		ConditionLevel: arg.ConditionLevel, TradeLocation: arg.TradeLocation, Status: arg.Status, ViewCount: arg.ViewCount}
	if s.products == nil {
		s.products = map[int64]gen.Product{}
	}
	s.products[100] = p
	return p, nil
}
func (s *stubQuerier) UpdateProduct(_ context.Context, arg gen.UpdateProductParams) (gen.Product, error) {
	p := s.products[arg.ID]
	p.Title, p.Description, p.CategoryID, p.Price, p.OriginalPrice = arg.Title, arg.Description, arg.CategoryID, arg.Price, arg.OriginalPrice
	p.ConditionLevel, p.TradeLocation, p.Status = arg.ConditionLevel, arg.TradeLocation, arg.Status
	s.products[arg.ID] = p
	return p, nil
}
func (s *stubQuerier) IncrementProductView(_ context.Context, id int64) error {
	s.viewIncFor = append(s.viewIncFor, id)
	return nil
}
func (s *stubQuerier) SetProductStatus(_ context.Context, arg gen.SetProductStatusParams) error {
	s.statusSet = append(s.statusSet, arg)
	return nil
}
func (s *stubQuerier) CountProducts(_ context.Context, arg gen.CountProductsParams) (int64, error) {
	s.countParams = arg
	return s.countResult, nil
}
func (s *stubQuerier) ListProducts(_ context.Context, arg gen.ListProductsParams) ([]gen.Product, error) {
	s.listParams = arg
	return s.listResult, nil
}
func (s *stubQuerier) ListProductImages(_ context.Context, _ []int64) ([]gen.ProductImage, error) {
	return s.images, nil
}
func (s *stubQuerier) InsertProductImage(_ context.Context, arg gen.InsertProductImageParams) error {
	s.insertImages = append(s.insertImages, arg)
	return nil
}
func (s *stubQuerier) DeleteProductImages(_ context.Context, productID int64) error {
	s.deletedImgFor = append(s.deletedImgFor, productID)
	return nil
}
func (s *stubQuerier) ListUsersByIDs(_ context.Context, _ []int64) ([]gen.ListUsersByIDsRow, error) {
	return s.users, nil
}
func (s *stubQuerier) ListCategoriesByIDs(_ context.Context, _ []int64) ([]gen.ListCategoriesByIDsRow, error) {
	return s.cats, nil
}
func (s *stubQuerier) ListFavoritedProductIDs(_ context.Context, _ gen.ListFavoritedProductIDsParams) ([]int64, error) {
	return s.favIDs, nil
}
func (s *stubQuerier) ListProductComments(_ context.Context, _ int64) ([]gen.ProductComment, error) {
	return s.comments, nil
}
func (s *stubQuerier) InsertProductComment(_ context.Context, arg gen.InsertProductCommentParams) (gen.ProductComment, error) {
	s.insertedComm = arg
	return gen.ProductComment{ID: 7, ProductID: arg.ProductID, UserID: arg.UserID, Content: arg.Content}, nil
}

// fakeTx is a no-op pgx.Tx used to exercise the transactional write paths
// without a real database. The service builds its tx-scoped writer via
// newTxWriter (overridden in newTestService to route to the stub), so this tx
// is never used for actual queries — only Begin/Commit/Rollback are invoked.
type fakeTx struct {
	pgx.Tx
	committed  bool
	rolledBack bool
}

func (t *fakeTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *fakeTx) Rollback(context.Context) error { t.rolledBack = true; return nil }

// fakeBeginner satisfies TxBeginner, handing out the same fakeTx each Begin.
type fakeBeginner struct {
	tx *fakeTx
}

func (b *fakeBeginner) Begin(context.Context) (pgx.Tx, error) {
	if b.tx == nil {
		b.tx = &fakeTx{}
	}
	return b.tx, nil
}

// newTestService wires a Service whose transactional writes route back to the
// stub, so Create/Update can be tested without a real DB.
func newTestService(stub *stubQuerier) *Service {
	svc := NewService(stub, &fakeBeginner{})
	svc.newTxWriter = func(pgx.Tx) txWriter { return stub }
	return svc
}

func num(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		t.Fatalf("scan %q: %v", s, err)
	}
	return n
}

func price(t *testing.T, s string) Price {
	t.Helper()
	return Price{num(t, s)}
}

// --- pagination math + defaults ---

func TestListPaginationDefaults(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	if _, err := svc.List(context.Background(), ListQuery{}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.listParams.Lim != 12 {
		t.Fatalf("default size want 12, got %d", stub.listParams.Lim)
	}
	if stub.listParams.Off != 0 {
		t.Fatalf("default page 1 offset want 0, got %d", stub.listParams.Off)
	}
}

func TestListPaginationOffset(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	page, size := int32(3), int32(20)
	if _, err := svc.List(context.Background(), ListQuery{Page: &page, Size: &size}, nil); err != nil {
		t.Fatal(err)
	}
	// 1-based page: offset = (3-1)*20 = 40
	if stub.listParams.Off != 40 {
		t.Fatalf("offset want 40, got %d", stub.listParams.Off)
	}
	if stub.listParams.Lim != 20 {
		t.Fatalf("limit want 20, got %d", stub.listParams.Lim)
	}
}

func TestListPaginationInvalidFallsBack(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	page, size := int32(0), int32(-5)
	if _, err := svc.List(context.Background(), ListQuery{Page: &page, Size: &size}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.listParams.Off != 0 || stub.listParams.Lim != 12 {
		t.Fatalf("invalid page/size want off=0 lim=12, got off=%d lim=%d", stub.listParams.Off, stub.listParams.Lim)
	}
}

// --- sort-key passthrough ---

func TestListSortByPassthrough(t *testing.T) {
	for _, sb := range []string{"price_asc", "price_desc", "hot", "newest", ""} {
		stub := &stubQuerier{}
		svc := newTestService(stub)
		var sbp *string
		if sb != "" {
			v := sb
			sbp = &v
		}
		if _, err := svc.List(context.Background(), ListQuery{SortBy: sbp}, nil); err != nil {
			t.Fatal(err)
		}
		if stub.listParams.SortBy != sb {
			t.Fatalf("sortBy want %q, got %q", sb, stub.listParams.SortBy)
		}
	}
}

// --- status tri-state logic (mirrors Java) ---

func TestListStatusExplicitDisablesDefault(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	st := "OFFLINE"
	if _, err := svc.List(context.Background(), ListQuery{Status: &st}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.listParams.Status == nil || *stub.listParams.Status != "OFFLINE" {
		t.Fatalf("explicit status not passed through: %v", stub.listParams.Status)
	}
	if stub.listParams.ApplyDefaultStatus {
		t.Fatal("apply default should be false when status explicit")
	}
}

func TestListDefaultStatusWhenNoSeller(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	if _, err := svc.List(context.Background(), ListQuery{}, nil); err != nil {
		t.Fatal(err)
	}
	if !stub.listParams.ApplyDefaultStatus {
		t.Fatal("apply default should be true when no seller and no includeAllStatus")
	}
}

func TestListSellerDisablesDefaultStatus(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	sid := int64(5)
	if _, err := svc.List(context.Background(), ListQuery{SellerID: &sid}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.listParams.ApplyDefaultStatus {
		t.Fatal("apply default should be false when sellerId set")
	}
}

func TestListIncludeAllStatusDisablesDefault(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	yes := true
	if _, err := svc.List(context.Background(), ListQuery{IncludeAllStatus: &yes}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.listParams.ApplyDefaultStatus {
		t.Fatal("apply default should be false when includeAllStatus=true")
	}
}

func TestListBlankKeywordBecomesNil(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	blank := "   "
	if _, err := svc.List(context.Background(), ListQuery{Keyword: &blank}, nil); err != nil {
		t.Fatal(err)
	}
	if stub.listParams.Keyword != nil {
		t.Fatalf("blank keyword should become nil, got %v", *stub.listParams.Keyword)
	}
}

// --- empty list serializes [] not null ---

func TestListEmptyRecordsSerializeArray(t *testing.T) {
	stub := &stubQuerier{countResult: 0, listResult: nil}
	svc := newTestService(stub)
	page, err := svc.List(context.Background(), ListQuery{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(httpx.OK(page))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"records":[]`) {
		t.Fatalf("empty records must serialize as []: %s", b)
	}
	if !strings.Contains(string(b), `"total":0`) {
		t.Fatalf("total must be 0: %s", b)
	}
}

// --- price JSON formatting matches Java BigDecimal (scale preserved) ---

func TestPriceJSONFormatMatchesJavaBigDecimal(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "12.50"), OriginalPrice: num(t, "99.00")},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	// Jackson serializes NUMERIC(10,2) BigDecimal preserving scale: 12.50, not 12.5
	if !strings.Contains(string(b), `"price":12.50`) {
		t.Fatalf("price format mismatch (want 12.50): %s", b)
	}
	if !strings.Contains(string(b), `"originalPrice":99.00`) {
		t.Fatalf("originalPrice format mismatch (want 99.00): %s", b)
	}
}

func TestPriceNullSerializesNull(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "5.00")},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(item)
	if !strings.Contains(string(b), `"originalPrice":null`) {
		t.Fatalf("null originalPrice must serialize null: %s", b)
	}
}

// --- view_count increment reflected in detail ---

func TestDetailIncrementsAndReflectsViewCount(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00"), ViewCount: 41},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.viewIncFor) != 1 || stub.viewIncFor[0] != 1 {
		t.Fatalf("expected one view increment for id 1, got %v", stub.viewIncFor)
	}
	if item.ViewCount != 42 {
		t.Fatalf("returned viewCount should reflect increment (42), got %d", item.ViewCount)
	}
}

func TestDetailNotFound(t *testing.T) {
	svc := newTestService(&stubQuerier{})
	_, err := svc.Detail(context.Background(), 999, nil)
	var be httpx.BizError
	if !errors.As(err, &be) || be.Msg != "商品不存在" {
		t.Fatalf("want 商品不存在 BizError, got %v", err)
	}
}

// --- ownership check message/code ---

func TestUpdateNonOwnerDenied(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00")},
		},
	}
	svc := newTestService(stub)
	_, err := svc.Update(context.Background(), 1, 999, false, UpdateReq{})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Code != 403 || be.Msg != "无权操作" {
		t.Fatalf("ownership denial want code=403 msg=无权操作, got code=%d msg=%q", be.Code, be.Msg)
	}
}

func TestUpdateAdminBypassesOwnership(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00")},
		},
	}
	svc := newTestService(stub)
	newTitle := "updated"
	_, err := svc.Update(context.Background(), 1, 999, true, UpdateReq{Title: &newTitle})
	if err != nil {
		t.Fatalf("admin should bypass ownership: %v", err)
	}
}

func TestChangeStatusNonOwnerDenied(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00")},
		},
	}
	svc := newTestService(stub)
	err := svc.ChangeStatus(context.Background(), 1, 999, false, "OFFLINE")
	var be httpx.BizError
	if !errors.As(err, &be) || be.Code != 403 || be.Msg != "无权操作" {
		t.Fatalf("want 403 无权操作, got %v", err)
	}
}

func TestChangeStatusOwnerSetsOffline(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00")},
		},
	}
	svc := newTestService(stub)
	if err := svc.ChangeStatus(context.Background(), 1, 2, false, "OFFLINE"); err != nil {
		t.Fatal(err)
	}
	if len(stub.statusSet) != 1 || stub.statusSet[0].Status != "OFFLINE" {
		t.Fatalf("expected status set to OFFLINE, got %v", stub.statusSet)
	}
}

// --- create validation + escaping + image sort order ---

func TestCreateValidation(t *testing.T) {
	svc := newTestService(&stubQuerier{})
	cid := int64(3)
	cases := []struct {
		name string
		req  CreateReq
		msg  string
	}{
		{"no title", CreateReq{CategoryID: &cid, Price: price(t, "1.00")}, "请输入标题"},
		{"no category", CreateReq{Title: "t", Price: price(t, "1.00")}, "请选择分类"},
		{"no price", CreateReq{Title: "t", CategoryID: &cid}, "请输入价格"},
		{"zero price", CreateReq{Title: "t", CategoryID: &cid, Price: price(t, "0.00")}, "价格必须大于0"},
		{"negative price", CreateReq{Title: "t", CategoryID: &cid, Price: price(t, "-5.00")}, "价格必须大于0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), 1, tc.req)
			var be httpx.BizError
			if !errors.As(err, &be) || be.Msg != tc.msg {
				t.Fatalf("want %q, got %v", tc.msg, err)
			}
		})
	}
}

func TestListPaginationOverflowClamped(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	// Absurd inputs that would overflow int32 offset math ((page-1)*size).
	page, size := int32(1<<30), int32(1<<20)
	if _, err := svc.List(context.Background(), ListQuery{Page: &page, Size: &size}, nil); err != nil {
		t.Fatal(err)
	}
	// size clamped to 100; offset clamped to max int32 (no overflow/negative).
	if stub.listParams.Lim != 100 {
		t.Fatalf("size should clamp to 100, got %d", stub.listParams.Lim)
	}
	if stub.listParams.Off < 0 {
		t.Fatalf("offset must not overflow negative, got %d", stub.listParams.Off)
	}
}

func TestCreateCommitsTransaction(t *testing.T) {
	stub := &stubQuerier{}
	beginner := &fakeBeginner{}
	svc := NewService(stub, beginner)
	svc.newTxWriter = func(pgx.Tx) txWriter { return stub }
	cid := int64(3)
	if _, err := svc.Create(context.Background(), 1, CreateReq{
		Title: "t", CategoryID: &cid, Price: price(t, "1.00"), Images: []string{"a.jpg"},
	}); err != nil {
		t.Fatal(err)
	}
	if beginner.tx == nil || !beginner.tx.committed {
		t.Fatal("Create must commit its transaction")
	}
}

func TestCreateEscapesAndOrdersImages(t *testing.T) {
	stub := &stubQuerier{}
	svc := newTestService(stub)
	cid := int64(3)
	desc := "<b>hi</b>"
	_, err := svc.Create(context.Background(), 1, CreateReq{
		Title:       "<t>",
		Description: &desc,
		CategoryID:  &cid,
		Price:       price(t, "1.00"),
		Images:      []string{"a.jpg", "b.jpg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	created := stub.products[100]
	if created.Title != "&lt;t&gt;" {
		t.Fatalf("title not escaped: %q", created.Title)
	}
	if created.Description == nil || *created.Description != "&lt;b&gt;hi&lt;/b&gt;" {
		t.Fatalf("description not escaped: %v", created.Description)
	}
	if len(stub.insertImages) != 2 || stub.insertImages[0].SortOrder != 0 || stub.insertImages[1].SortOrder != 1 {
		t.Fatalf("images not inserted with 0-based sort order: %+v", stub.insertImages)
	}
}

// --- enrich: cover, images [], favorited, seller/category ---

func TestEnrichCoverAndFavorited(t *testing.T) {
	campus := "东校区"
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00")},
		},
		images: []gen.ProductImage{
			{ProductID: 1, ImageUrl: "cover.jpg", SortOrder: 0},
			{ProductID: 1, ImageUrl: "second.jpg", SortOrder: 1},
		},
		users:  []gen.ListUsersByIDsRow{{ID: 2, Name: "Alice", Campus: &campus, StudentNo: "20210001"}},
		cats:   []gen.ListCategoriesByIDsRow{{ID: 3, Name: "书籍"}},
		favIDs: []int64{1},
	}
	svc := newTestService(stub)
	uid := int64(9)
	item, err := svc.Detail(context.Background(), 1, &uid)
	if err != nil {
		t.Fatal(err)
	}
	if item.Cover == nil || *item.Cover != "cover.jpg" {
		t.Fatalf("cover want cover.jpg, got %v", item.Cover)
	}
	if len(item.Images) != 2 {
		t.Fatalf("want 2 images, got %d", len(item.Images))
	}
	if !item.Favorited {
		t.Fatal("expected favorited true")
	}
	if item.SellerName == nil || *item.SellerName != "Alice" {
		t.Fatalf("seller name mismatch: %v", item.SellerName)
	}
	if item.CategoryName == nil || *item.CategoryName != "书籍" {
		t.Fatalf("category name mismatch: %v", item.CategoryName)
	}
}

func TestEnrichNoImagesGivesNullCoverEmptyArray(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{
			1: {ID: 1, SellerID: 2, CategoryID: 3, Title: "x", Price: num(t, "1.00")},
		},
	}
	svc := newTestService(stub)
	item, err := svc.Detail(context.Background(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if item.Cover != nil {
		t.Fatalf("cover should be nil, got %v", *item.Cover)
	}
	b, _ := json.Marshal(item)
	if !strings.Contains(string(b), `"images":[]`) {
		t.Fatalf("images must serialize as []: %s", b)
	}
	if !strings.Contains(string(b), `"cover":null`) {
		t.Fatalf("cover must serialize null: %s", b)
	}
}

// --- comments ---

func TestStudentNoSuffix(t *testing.T) {
	cases := map[string]string{
		"20210001": "0001",
		"123":      "123",
		"":         "",
		"   ":      "",
		"ABCDEF":   "CDEF",
	}
	for in, want := range cases {
		if got := studentNoSuffix(in); got != want {
			t.Fatalf("studentNoSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCreateCommentValidation(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{1: {ID: 1, SellerID: 2, CategoryID: 3, Price: num(t, "1.00")}},
	}
	svc := newTestService(stub)

	_, err := svc.CreateComment(context.Background(), 1, 9, CommentCreateReq{Content: "   "})
	var be httpx.BizError
	if !errors.As(err, &be) || be.Msg != "留言内容不能为空" {
		t.Fatalf("want 留言内容不能为空, got %v", err)
	}

	_, err = svc.CreateComment(context.Background(), 1, 9, CommentCreateReq{Content: strings.Repeat("x", 301)})
	if !errors.As(err, &be) || be.Msg != "留言最多 300 字" {
		t.Fatalf("want 留言最多 300 字, got %v", err)
	}
}

func TestCreateCommentEscapesAndReturnsCommenter(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{1: {ID: 1, SellerID: 2, CategoryID: 3, Price: num(t, "1.00")}},
		users:    []gen.ListUsersByIDsRow{{ID: 9, Name: "Bob", StudentNo: "20219999"}},
	}
	svc := newTestService(stub)
	item, err := svc.CreateComment(context.Background(), 1, 9, CommentCreateReq{Content: "<hi>"})
	if err != nil {
		t.Fatal(err)
	}
	if stub.insertedComm.Content != "&lt;hi&gt;" {
		t.Fatalf("content not escaped: %q", stub.insertedComm.Content)
	}
	if item.UserName == nil || *item.UserName != "Bob" {
		t.Fatalf("commenter name mismatch: %v", item.UserName)
	}
	if item.StudentNoSuffix == nil || *item.StudentNoSuffix != "9999" {
		t.Fatalf("student suffix mismatch: %v", item.StudentNoSuffix)
	}
}

func TestCommentsProductNotFound(t *testing.T) {
	svc := newTestService(&stubQuerier{})
	_, err := svc.ListComments(context.Background(), 404)
	var be httpx.BizError
	if !errors.As(err, &be) || be.Msg != "商品不存在" {
		t.Fatalf("want 商品不存在, got %v", err)
	}
}
