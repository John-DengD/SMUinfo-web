package product

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

// TxBeginner begins a database transaction. Satisfied by *pgxpool.Pool.
// Kept as a narrow interface so tests that don't exercise the transactional
// write paths (Create/Update) can leave it nil.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Querier is the subset of the sqlc-generated *gen.Queries the product service needs.
type Querier interface {
	GetProduct(ctx context.Context, id int64) (gen.Product, error)
	InsertProduct(ctx context.Context, arg gen.InsertProductParams) (gen.Product, error)
	UpdateProduct(ctx context.Context, arg gen.UpdateProductParams) (gen.Product, error)
	IncrementProductView(ctx context.Context, id int64) error
	SetProductStatus(ctx context.Context, arg gen.SetProductStatusParams) error
	CountProducts(ctx context.Context, arg gen.CountProductsParams) (int64, error)
	ListProducts(ctx context.Context, arg gen.ListProductsParams) ([]gen.Product, error)
	ListProductImages(ctx context.Context, productIds []int64) ([]gen.ProductImage, error)
	InsertProductImage(ctx context.Context, arg gen.InsertProductImageParams) error
	DeleteProductImages(ctx context.Context, productID int64) error
	ListUsersByIDs(ctx context.Context, ids []int64) ([]gen.ListUsersByIDsRow, error)
	ListCategoriesByIDs(ctx context.Context, ids []int64) ([]gen.ListCategoriesByIDsRow, error)
	ListFavoritedProductIDs(ctx context.Context, arg gen.ListFavoritedProductIDsParams) ([]int64, error)
	ListProductComments(ctx context.Context, productID int64) ([]gen.ProductComment, error)
	InsertProductComment(ctx context.Context, arg gen.InsertProductCommentParams) (gen.ProductComment, error)
}

// txWriter is the set of write operations Create/Update run inside a
// transaction. Both *gen.Queries (production) and the test stub satisfy it.
type txWriter interface {
	InsertProduct(ctx context.Context, arg gen.InsertProductParams) (gen.Product, error)
	UpdateProduct(ctx context.Context, arg gen.UpdateProductParams) (gen.Product, error)
	InsertProductImage(ctx context.Context, arg gen.InsertProductImageParams) error
	DeleteProductImages(ctx context.Context, productID int64) error
}

type Service struct {
	q    Querier
	pool TxBeginner
	// newTxWriter builds the tx-scoped writer from a begun transaction. In
	// production it returns gen.New(tx); tests can override it to route tx
	// writes back to their stub without a real DB.
	newTxWriter func(tx pgx.Tx) txWriter
}

// NewService constructs the product service. pool is used only by the
// multi-write paths (Create/Update) to run them atomically; read-only and
// single-write paths go through q. In production both point at the same
// *pgxpool.Pool (q via gen.New(pool)); tests that don't touch Create/Update
// may pass a nil pool.
func NewService(q Querier, pool TxBeginner) *Service {
	return &Service{
		q:           q,
		pool:        pool,
		newTxWriter: func(tx pgx.Tx) txWriter { return gen.New(tx) },
	}
}

// Item mirrors ProductDTO.Item (camelCase wire contract). price/originalPrice
// use the Price type, which serializes at fixed scale 2 (e.g. 12.50, 99.00),
// matching Java Jackson BigDecimal-from-NUMERIC(10,2) serialization byte-for-byte.
// When null they serialize to JSON null (matching a null BigDecimal in Java).
type Item struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Description    *string    `json:"description"`
	Price          Price      `json:"price"`
	OriginalPrice  Price      `json:"originalPrice"`
	ConditionLevel *string    `json:"conditionLevel"`
	TradeLocation  *string    `json:"tradeLocation"`
	Status         string     `json:"status"`
	ViewCount      int32      `json:"viewCount"`
	CreatedAt      *time.Time `json:"createdAt"`
	CategoryID     int64      `json:"categoryId"`
	CategoryName   *string    `json:"categoryName"`
	SellerID       int64      `json:"sellerId"`
	SellerName     *string    `json:"sellerName"`
	SellerCampus   *string    `json:"sellerCampus"`
	Images         []string   `json:"images"`
	Cover          *string    `json:"cover"`
	Favorited      bool       `json:"favorited"`
}

// CreateReq mirrors ProductDTO.CreateReq.
type CreateReq struct {
	Title          string   `json:"title"`
	Description    *string  `json:"description"`
	CategoryID     *int64   `json:"categoryId"`
	Price          Price    `json:"price"`
	OriginalPrice  Price    `json:"originalPrice"`
	ConditionLevel *string  `json:"conditionLevel"`
	TradeLocation  *string  `json:"tradeLocation"`
	Images         []string `json:"images"`
}

// UpdateReq mirrors ProductDTO.UpdateReq. All fields optional; pointer/Valid
// distinguishes "provided" from "absent" to match Java's `if (req.getX() != null)`.
type UpdateReq struct {
	Title          *string   `json:"title"`
	Description    *string   `json:"description"`
	CategoryID     *int64    `json:"categoryId"`
	Price          Price     `json:"price"`
	OriginalPrice  Price     `json:"originalPrice"`
	ConditionLevel *string   `json:"conditionLevel"`
	TradeLocation  *string   `json:"tradeLocation"`
	Status         *string   `json:"status"`
	Images         *[]string `json:"images"`
}

// ListQuery mirrors ProductDTO.ListQuery.
type ListQuery struct {
	Keyword          *string
	CategoryID       *int64
	MinPrice         pgtype.Numeric
	MaxPrice         pgtype.Numeric
	ConditionLevel   *string
	Campus           *string
	SortBy           *string
	Status           *string
	SellerID         *int64
	IncludeAllStatus *bool
	Page             *int32
	Size             *int32
}

func timePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

// List replicates ProductService.list: multi-condition search, sort, pagination.
func (s *Service) List(ctx context.Context, q ListQuery, currentUserID *int64) (httpx.Page, error) {
	// page defaults to 1, size defaults to 12 (Java defaults).
	page := int32(1)
	if q.Page != nil {
		page = *q.Page
	}
	size := int32(12)
	if q.Size != nil {
		size = *q.Size
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 12
	}
	// Clamp size to a sane maximum (Java default is 12; happy path unaffected)
	// and compute the offset in int64 so absurd query values can't overflow the
	// int32 multiplication before it is stored back in an int32 SQL parameter.
	if size > 100 {
		size = 100
	}
	off64 := (int64(page) - 1) * int64(size)
	if off64 > int64(^uint32(0)>>1) { // clamp to max int32
		off64 = int64(^uint32(0) >> 1)
	}
	offset := int32(off64)

	// Status logic mirrors Java: explicit status filters directly; otherwise, when
	// no sellerId and includeAllStatus is not true, restrict to ON_SALE/RESERVED.
	var status *string
	applyDefault := false
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		status = q.Status
	} else if q.SellerID == nil && (q.IncludeAllStatus == nil || !*q.IncludeAllStatus) {
		applyDefault = true
	}

	keyword := blankToNil(q.Keyword)
	conditionLevel := blankToNil(q.ConditionLevel)
	campus := blankToNil(q.Campus)
	sortBy := ""
	if q.SortBy != nil {
		sortBy = *q.SortBy
	}

	total, err := s.q.CountProducts(ctx, gen.CountProductsParams{
		Keyword:            keyword,
		CategoryID:         q.CategoryID,
		MinPrice:           q.MinPrice,
		MaxPrice:           q.MaxPrice,
		ConditionLevel:     conditionLevel,
		Campus:             campus,
		SellerID:           q.SellerID,
		Status:             status,
		ApplyDefaultStatus: applyDefault,
	})
	if err != nil {
		return httpx.Page{}, err
	}

	rows, err := s.q.ListProducts(ctx, gen.ListProductsParams{
		Keyword:            keyword,
		CategoryID:         q.CategoryID,
		MinPrice:           q.MinPrice,
		MaxPrice:           q.MaxPrice,
		ConditionLevel:     conditionLevel,
		Campus:             campus,
		SellerID:           q.SellerID,
		Status:             status,
		ApplyDefaultStatus: applyDefault,
		SortBy:             sortBy,
		Off:                offset,
		Lim:                size,
	})
	if err != nil {
		return httpx.Page{}, err
	}

	items, err := s.enrich(ctx, rows, currentUserID)
	if err != nil {
		return httpx.Page{}, err
	}
	return httpx.Page{Total: total, Records: items}, nil
}

// Detail replicates ProductService.detail: increments view_count then returns
// the enriched item (with the incremented count reflected).
func (s *Service) Detail(ctx context.Context, id int64, currentUserID *int64) (Item, error) {
	p, err := s.q.GetProduct(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, httpx.Biz("商品不存在")
		}
		return Item{}, err
	}
	if err := s.q.IncrementProductView(ctx, id); err != nil {
		return Item{}, err
	}
	p.ViewCount++ // reflect the increment in the returned payload (matches Java)
	items, err := s.enrich(ctx, []gen.Product{p}, currentUserID)
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

// Create replicates ProductService.create.
func (s *Service) Create(ctx context.Context, sellerID int64, req CreateReq) (Item, error) {
	if strings.TrimSpace(req.Title) == "" {
		return Item{}, httpx.Biz("请输入标题")
	}
	if req.CategoryID == nil {
		return Item{}, httpx.Biz("请选择分类")
	}
	if !req.Price.Valid {
		return Item{}, httpx.Biz("请输入价格")
	}
	// Mirror Java @Positive on price: reject price <= 0 as a business error.
	if !priceIsPositive(req.Price) {
		return Item{}, httpx.Biz("价格必须大于0")
	}

	title := html.EscapeString(req.Title)
	var desc *string
	if req.Description != nil {
		d := html.EscapeString(*req.Description)
		desc = &d
	}

	// Product insert + image inserts must be atomic (matches Java @Transactional):
	// a mid-operation failure must not leave a product row without its images.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Item{}, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit
	qtx := s.newTxWriter(tx)

	p, err := qtx.InsertProduct(ctx, gen.InsertProductParams{
		SellerID:       sellerID,
		CategoryID:     *req.CategoryID,
		Title:          title,
		Description:    desc,
		Price:          req.Price.Numeric,
		OriginalPrice:  req.OriginalPrice.Numeric,
		ConditionLevel: req.ConditionLevel,
		TradeLocation:  req.TradeLocation,
		Status:         "ON_SALE",
		ViewCount:      0,
	})
	if err != nil {
		return Item{}, err
	}
	if err := saveImages(ctx, qtx, p.ID, req.Images); err != nil {
		return Item{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Item{}, err
	}
	// Java returns detail(p.getId(), sellerId), which increments view_count.
	return s.Detail(ctx, p.ID, &sellerID)
}

// Update replicates ProductService.update: owner-or-admin only, partial update,
// optional full image replacement.
func (s *Service) Update(ctx context.Context, productID, currentUserID int64, isAdmin bool, req UpdateReq) (Item, error) {
	p, err := s.q.GetProduct(ctx, productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, httpx.Biz("商品不存在")
		}
		return Item{}, err
	}
	if !isAdmin && p.SellerID != currentUserID {
		return Item{}, httpx.NewBiz(403, "无权操作")
	}

	if req.Title != nil {
		p.Title = html.EscapeString(*req.Title)
	}
	if req.Description != nil {
		d := html.EscapeString(*req.Description)
		p.Description = &d
	}
	if req.CategoryID != nil {
		p.CategoryID = *req.CategoryID
	}
	if req.Price.Valid {
		p.Price = req.Price.Numeric
	}
	if req.OriginalPrice.Valid {
		p.OriginalPrice = req.OriginalPrice.Numeric
	}
	if req.ConditionLevel != nil {
		p.ConditionLevel = req.ConditionLevel
	}
	if req.TradeLocation != nil {
		p.TradeLocation = req.TradeLocation
	}
	if req.Status != nil {
		p.Status = *req.Status
	}

	// Product update + (optional) full image replacement must be atomic
	// (matches Java @Transactional): a failure between deleting the old images
	// and re-inserting the new ones must not wipe the product's image state.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Item{}, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit
	qtx := s.newTxWriter(tx)

	if _, err := qtx.UpdateProduct(ctx, gen.UpdateProductParams{
		ID:             p.ID,
		Title:          p.Title,
		Description:    p.Description,
		CategoryID:     p.CategoryID,
		Price:          p.Price,
		OriginalPrice:  p.OriginalPrice,
		ConditionLevel: p.ConditionLevel,
		TradeLocation:  p.TradeLocation,
		Status:         p.Status,
	}); err != nil {
		return Item{}, err
	}

	if req.Images != nil {
		if err := qtx.DeleteProductImages(ctx, productID); err != nil {
			return Item{}, err
		}
		if err := saveImages(ctx, qtx, productID, *req.Images); err != nil {
			return Item{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Item{}, err
	}
	// Java returns detail(productId, currentUserId), which increments view_count.
	return s.Detail(ctx, productID, &currentUserID)
}

// ChangeStatus replicates ProductService.changeStatus (used by DELETE → OFFLINE).
func (s *Service) ChangeStatus(ctx context.Context, productID, currentUserID int64, isAdmin bool, status string) error {
	p, err := s.q.GetProduct(ctx, productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("商品不存在")
		}
		return err
	}
	if !isAdmin && p.SellerID != currentUserID {
		return httpx.NewBiz(403, "无权操作")
	}
	return s.q.SetProductStatus(ctx, gen.SetProductStatusParams{ID: productID, Status: status})
}

// imageInserter is the single-method subset saveImages needs; satisfied both by
// the *gen.Queries built from a tx and by the base Querier.
type imageInserter interface {
	InsertProductImage(ctx context.Context, arg gen.InsertProductImageParams) error
}

func saveImages(ctx context.Context, q imageInserter, productID int64, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	for i, url := range urls {
		if err := q.InsertProductImage(ctx, gen.InsertProductImageParams{
			ProductID: productID,
			ImageUrl:  url,
			SortOrder: int32(i),
		}); err != nil {
			return err
		}
	}
	return nil
}

// priceIsPositive reports whether a valid Price is strictly greater than zero,
// mirroring Java's @Positive on ProductDTO.CreateReq.price.
func priceIsPositive(p Price) bool {
	if !p.Valid || p.NaN || p.Int == nil {
		return false
	}
	return p.Int.Sign() > 0
}

// enrich replicates ProductService.enrich: batch-load images, sellers, categories,
// and (when logged in) the current user's favorites.
func (s *Service) enrich(ctx context.Context, list []gen.Product, currentUserID *int64) ([]Item, error) {
	if len(list) == 0 {
		return make([]Item, 0), nil
	}

	productIDs := make([]int64, 0, len(list))
	sellerSet := map[int64]struct{}{}
	categorySet := map[int64]struct{}{}
	for _, p := range list {
		productIDs = append(productIDs, p.ID)
		sellerSet[p.SellerID] = struct{}{}
		categorySet[p.CategoryID] = struct{}{}
	}

	imgs, err := s.q.ListProductImages(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	imageMap := map[int64][]string{}
	for _, img := range imgs {
		imageMap[img.ProductID] = append(imageMap[img.ProductID], img.ImageUrl)
	}

	users, err := s.q.ListUsersByIDs(ctx, keys(sellerSet))
	if err != nil {
		return nil, err
	}
	userMap := map[int64]gen.ListUsersByIDsRow{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	cats, err := s.q.ListCategoriesByIDs(ctx, keys(categorySet))
	if err != nil {
		return nil, err
	}
	catMap := map[int64]string{}
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}

	favorited := map[int64]struct{}{}
	if currentUserID != nil {
		favIDs, err := s.q.ListFavoritedProductIDs(ctx, gen.ListFavoritedProductIDsParams{
			UserID:     *currentUserID,
			ProductIds: productIDs,
		})
		if err != nil {
			return nil, err
		}
		for _, id := range favIDs {
			favorited[id] = struct{}{}
		}
	}

	result := make([]Item, 0, len(list))
	for _, p := range list {
		it := Item{
			ID:             p.ID,
			Title:          p.Title,
			Description:    p.Description,
			Price:          Price{p.Price},
			OriginalPrice:  Price{p.OriginalPrice},
			ConditionLevel: p.ConditionLevel,
			TradeLocation:  p.TradeLocation,
			Status:         p.Status,
			ViewCount:      p.ViewCount,
			CreatedAt:      timePtr(p.CreatedAt),
			CategoryID:     p.CategoryID,
			SellerID:       p.SellerID,
		}
		if name, ok := catMap[p.CategoryID]; ok {
			n := name
			it.CategoryName = &n
		}
		if u, ok := userMap[p.SellerID]; ok {
			n := u.Name
			it.SellerName = &n
			it.SellerCampus = u.Campus
		}
		imgList := imageMap[p.ID]
		it.Images = make([]string, 0, len(imgList))
		it.Images = append(it.Images, imgList...)
		if len(imgList) > 0 {
			cover := imgList[0]
			it.Cover = &cover
		}
		if _, ok := favorited[p.ID]; ok {
			it.Favorited = true
		}
		result = append(result, it)
	}
	return result, nil
}

// --- comments ---

// Comment mirrors ProductCommentDTO.Item.
type Comment struct {
	ID              int64      `json:"id"`
	ProductID       int64      `json:"productId"`
	UserID          int64      `json:"userId"`
	UserName        *string    `json:"userName"`
	StudentNoSuffix *string    `json:"studentNoSuffix"`
	Content         string     `json:"content"`
	CreatedAt       *time.Time `json:"createdAt"`
}

// CommentCreateReq mirrors ProductCommentDTO.CreateReq.
type CommentCreateReq struct {
	Content string `json:"content"`
}

// ListComments replicates ProductCommentService.list.
func (s *Service) ListComments(ctx context.Context, productID int64) ([]Comment, error) {
	if err := s.ensureProduct(ctx, productID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListProductComments(ctx, productID)
	if err != nil {
		return nil, err
	}
	return s.enrichComments(ctx, rows)
}

// CreateComment replicates ProductCommentService.create.
func (s *Service) CreateComment(ctx context.Context, productID, userID int64, req CommentCreateReq) (Comment, error) {
	if err := s.ensureProduct(ctx, productID); err != nil {
		return Comment{}, err
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return Comment{}, httpx.Biz("留言内容不能为空")
	}
	if len([]rune(content)) > 300 {
		return Comment{}, httpx.Biz("留言最多 300 字")
	}
	saved, err := s.q.InsertProductComment(ctx, gen.InsertProductCommentParams{
		ProductID: productID,
		UserID:    userID,
		Content:   html.EscapeString(content),
	})
	if err != nil {
		return Comment{}, err
	}
	items, err := s.enrichComments(ctx, []gen.ProductComment{saved})
	if err != nil {
		return Comment{}, err
	}
	return items[0], nil
}

func (s *Service) ensureProduct(ctx context.Context, productID int64) error {
	if _, err := s.q.GetProduct(ctx, productID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("商品不存在")
		}
		return err
	}
	return nil
}

func (s *Service) enrichComments(ctx context.Context, comments []gen.ProductComment) ([]Comment, error) {
	if len(comments) == 0 {
		return make([]Comment, 0), nil
	}
	userSet := map[int64]struct{}{}
	for _, c := range comments {
		userSet[c.UserID] = struct{}{}
	}
	users, err := s.q.ListUsersByIDs(ctx, keys(userSet))
	if err != nil {
		return nil, err
	}
	userMap := map[int64]gen.ListUsersByIDsRow{}
	for _, u := range users {
		userMap[u.ID] = u
	}
	result := make([]Comment, 0, len(comments))
	for _, c := range comments {
		it := Comment{
			ID:        c.ID,
			ProductID: c.ProductID,
			UserID:    c.UserID,
			Content:   c.Content,
			CreatedAt: timePtr(c.CreatedAt),
		}
		if u, ok := userMap[c.UserID]; ok {
			n := u.Name
			it.UserName = &n
			suffix := studentNoSuffix(u.StudentNo)
			it.StudentNoSuffix = &suffix
		}
		result = append(result, it)
	}
	return result, nil
}

func studentNoSuffix(studentNo string) string {
	if strings.TrimSpace(studentNo) == "" {
		return ""
	}
	r := []rune(studentNo)
	start := len(r) - 4
	if start < 0 {
		start = 0
	}
	return string(r[start:])
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
