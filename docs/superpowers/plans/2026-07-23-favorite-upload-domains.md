# Favorite & Upload Domains Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `favorite` (Task 11) and `upload` (Task 19) domains for the SMU Deal Go/Gin/PostgreSQL rewrite, matching the Java wire contract exactly.

**Architecture:** Each domain lives under `server/internal/{domain}/` with handler.go (Gin routes), service.go (business logic + DTO types), and service_test.go (unit tests against stub Querier). Favorite uses sqlc-generated queries in `favorite.sql`; upload is pure filesystem logic needing no DB queries. Both domains register behind the `RequireAuth()` middleware group in main.go.

**Tech Stack:** Go 1.26, Gin v1.12, pgx/v5, sqlc v1.31.1, standard library `mime/multipart`, `path/filepath`, `os`, `crypto/rand` (for UUID-like hex filename generation — no external uuid dep needed).

---

## Key Contracts (from Java source)

### Favorite
- `GET /api/favorites` → returns `PageResult<ProductDTO.Item>` = `{"total": N, "records": [...ProductItem...]}` — **product items with full enrichment**, `favorited` always `true`
- `POST /api/favorites/{productId}` → idempotent add; if product not found → `BusinessException("商品不存在")`; if already favorited → silently return ok (Java checks count > 0, returns early)
- `DELETE /api/favorites/{productId}` → always ok (no error if not favorited)
- All three require auth.

### Upload
- `POST /api/upload/image` multipart field name: **`file`**
- Allowed extensions (lowercase): `jpg`, `jpeg`, `png`, `gif`, `webp`
- Max size: `cfg.MaxFileSize` (default 5 MB = 5242880 bytes); Java error message: `"文件不能超过 5MB"`
- On-disk path: `{UploadDir}/{yyyy-MM-dd}/{uuid_hex}.{ext}` — uuid_hex is 32-char hex (Java does `UUID.randomUUID().toString().replace("-","")`)
- Response JSON: `{"url": "/uploads/2026-07-23/abc123.jpg"}` — field name is `url`
- URL value = `cfg.URLPrefix + "/" + dateDir + "/" + filename` = `/uploads/2026-07-23/abc123.jpg`
- Error messages: `"文件不能为空"`, `"文件不能超过 5MB"`, `"文件名非法"`, `"仅支持 jpg/jpeg/png/gif/webp"`, `"创建目录失败"`, `"保存文件失败"`

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `server/internal/db/queries/favorite.sql` | Create | SQL queries: InsertFavorite, DeleteFavorite, CountFavorite, ListFavoriteProductIDs |
| `server/internal/db/gen/favorite.sql.go` | Generated | sqlc output — do not edit manually |
| `server/internal/favorite/service.go` | Create | Querier interface, Service struct, DTOs (reuses product.Item shape), Add/Remove/MyFavorites methods |
| `server/internal/favorite/handler.go` | Create | Register() wiring 3 routes behind RequireAuth |
| `server/internal/favorite/service_test.go` | Create | Unit tests: add idempotent, add product-not-found, remove, list |
| `server/internal/upload/service.go` | Create | Service struct, Upload() method with validation + file write |
| `server/internal/upload/handler.go` | Create | Register() wiring POST /upload/image behind RequireAuth |
| `server/internal/upload/service_test.go` | Create | Unit tests: extension validation, size validation, path scheme, file write to temp dir |
| `server/cmd/smudeal/main.go` | Modify | Import + register favorite and upload domains |

---

## Task 1: Write favorite SQL queries and run sqlc generate

**Files:**
- Create: `server/internal/db/queries/favorite.sql`
- Generated: `server/internal/db/gen/favorite.sql.go`

- [ ] **Step 1: Write favorite.sql**

```sql
-- name: InsertFavorite :exec
INSERT INTO favorite (user_id, product_id)
VALUES ($1, $2);

-- name: DeleteFavorite :exec
DELETE FROM favorite WHERE user_id = $1 AND product_id = $2;

-- name: CountFavorite :one
SELECT count(*) FROM favorite WHERE user_id = $1 AND product_id = $2;

-- name: ListFavoriteProductIDs :many
SELECT product_id FROM favorite
WHERE user_id = $1
ORDER BY created_at DESC;
```

Save to: `server/internal/db/queries/favorite.sql`

- [ ] **Step 2: Run sqlc generate**

```bash
cd server && sqlc generate
```

Expected: No errors. `server/internal/db/gen/favorite.sql.go` appears.

- [ ] **Step 3: Verify generated file**

```bash
grep -n "InsertFavorite\|DeleteFavorite\|CountFavorite\|ListFavoriteProductIDs" server/internal/db/gen/favorite.sql.go
```

Expected: All 4 function names appear.

---

## Task 2: Write favorite service

**Files:**
- Create: `server/internal/favorite/service.go`

The service reuses `product.Service.enrich()` indirectly — but since `enrich` is unexported and in a different package, the favorite service calls the product service's `List` logic differently. Actually, Java's `myFavorites` manually assembles product items; we'll replicate that by calling our own DB queries plus the same enrichment pattern from product. The cleanest approach that stays DRY with the existing product domain is to have the favorite service accept a product querier and do enrichment inline (same as Java does).

- [ ] **Step 1: Write service.go**

```go
package favorite

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
	"github.com/John-DengD/smu-deal/server/internal/product"
)

// Querier is the subset of sqlc-generated *gen.Queries the favorite service needs.
type Querier interface {
	// favorite queries
	InsertFavorite(ctx context.Context, arg gen.InsertFavoriteParams) error
	DeleteFavorite(ctx context.Context, arg gen.DeleteFavoriteParams) error
	CountFavorite(ctx context.Context, arg gen.CountFavoriteParams) (int64, error)
	ListFavoriteProductIDs(ctx context.Context, userID int64) ([]int64, error)
	// product enrichment queries (reused from product domain)
	GetProduct(ctx context.Context, id int64) (gen.Product, error)
	ListProductsByIDs(ctx context.Context, ids []int64) ([]gen.Product, error)
	ListProductImages(ctx context.Context, productIds []int64) ([]gen.ProductImage, error)
	ListUsersByIDs(ctx context.Context, ids []int64) ([]gen.ListUsersByIDsRow, error)
	ListCategoriesByIDs(ctx context.Context, ids []int64) ([]gen.ListCategoriesByIDsRow, error)
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

// Add adds a favorite idempotently. Returns 商品不存在 if product not found.
// Matches Java: check product exists, check count > 0 (return silently if dup), insert.
func (s *Service) Add(ctx context.Context, userID, productID int64) error {
	if _, err := s.q.GetProduct(ctx, productID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("商品不存在")
		}
		return err
	}
	count, err := s.q.CountFavorite(ctx, gen.CountFavoriteParams{UserID: userID, ProductID: productID})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // idempotent: already favorited
	}
	return s.q.InsertFavorite(ctx, gen.InsertFavoriteParams{UserID: userID, ProductID: productID})
}

// Remove removes a favorite. Always succeeds (no error if not favorited).
func (s *Service) Remove(ctx context.Context, userID, productID int64) error {
	return s.q.DeleteFavorite(ctx, gen.DeleteFavoriteParams{UserID: userID, ProductID: productID})
}

// MyFavorites returns the current user's favorited products as enriched product items.
// Matches Java FavoriteService.myFavorites: returns PageResult<ProductDTO.Item>.
func (s *Service) MyFavorites(ctx context.Context, userID int64) (httpx.Page, error) {
	productIDs, err := s.q.ListFavoriteProductIDs(ctx, userID)
	if err != nil {
		return httpx.Page{}, err
	}
	if len(productIDs) == 0 {
		return httpx.Page{Total: 0, Records: []product.Item{}}, nil
	}

	products, err := s.q.ListProductsByIDs(ctx, productIDs)
	if err != nil {
		return httpx.Page{}, err
	}

	// Build lookup maps
	pmap := map[int64]gen.Product{}
	sellerSet := map[int64]struct{}{}
	categorySet := map[int64]struct{}{}
	for _, p := range products {
		pmap[p.ID] = p
		sellerSet[p.SellerID] = struct{}{}
		categorySet[p.CategoryID] = struct{}{}
	}

	imgs, err := s.q.ListProductImages(ctx, productIDs)
	if err != nil {
		return httpx.Page{}, err
	}
	imageMap := map[int64][]string{}
	for _, img := range imgs {
		imageMap[img.ProductID] = append(imageMap[img.ProductID], img.ImageUrl)
	}

	users, err := s.q.ListUsersByIDs(ctx, keys(sellerSet))
	if err != nil {
		return httpx.Page{}, err
	}
	userMap := map[int64]gen.ListUsersByIDsRow{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	cats, err := s.q.ListCategoriesByIDs(ctx, keys(categorySet))
	if err != nil {
		return httpx.Page{}, err
	}
	catMap := map[int64]string{}
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}

	// Assemble items in favorite order (productIDs is already ordered by created_at DESC)
	items := make([]product.Item, 0, len(productIDs))
	for _, pid := range productIDs {
		p, ok := pmap[pid]
		if !ok {
			continue // product deleted since favorited
		}
		it := product.Item{
			ID:             p.ID,
			Title:          p.Title,
			Description:    p.Description,
			Price:          product.Price{Numeric: p.Price},
			OriginalPrice:  product.Price{Numeric: p.OriginalPrice},
			ConditionLevel: p.ConditionLevel,
			TradeLocation:  p.TradeLocation,
			Status:         p.Status,
			ViewCount:      p.ViewCount,
			CreatedAt:      timePtr(p),
			CategoryID:     p.CategoryID,
			SellerID:       p.SellerID,
			Favorited:      true,
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
		items = append(items, it)
	}
	return httpx.Page{Total: int64(len(items)), Records: items}, nil
}

func timePtr(p gen.Product) *time.Time {
	if !p.CreatedAt.Valid {
		return nil
	}
	t := p.CreatedAt.Time
	return &t
}

func keys(m map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
```

NOTE: `product.Item` and `product.Price` are exported from the product package — the favorite service imports them directly to avoid duplicating the wire shape. The `product.Price` struct has a `Numeric pgtype.Numeric` field (see `server/internal/product/price.go`) — check what the field is actually named before writing (it may be embedded or named).

- [ ] **Step 2: Check product.Price definition**

```bash
cat server/internal/product/price.go
```

Adjust the `product.Price{...}` construction in service.go to match the actual field layout.

- [ ] **Step 3: Check if ListProductsByIDs exists in gen**

```bash
grep -rn "ListProductsByIDs\|SelectBatchIds" server/internal/db/gen/ 2>/dev/null || echo "not found"
```

If not found: add `ListProductsByIDs` query to `favorite.sql` and re-run `sqlc generate`.

Query to add to favorite.sql:
```sql
-- name: ListProductsByIDs :many
SELECT id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at
FROM product WHERE id = ANY(sqlc.arg('ids')::bigint[]);
```

---

## Task 3: Write favorite handler

**Files:**
- Create: `server/internal/favorite/handler.go`

- [ ] **Step 1: Write handler.go**

```go
package favorite

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the favorite routes onto the /api group.
// All routes require authentication.
//
//	GET    /api/favorites              list current user's favorited products
//	POST   /api/favorites/:productId   add favorite (idempotent)
//	DELETE /api/favorites/:productId   remove favorite
func Register(api *gin.RouterGroup, svc *Service) {
	fav := api.Group("/favorites")
	fav.Use(httpx.RequireAuth())
	{
		fav.GET("", func(c *gin.Context) {
			page, err := svc.MyFavorites(c.Request.Context(), httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(page))
		})

		fav.POST("/:productId", func(c *gin.Context) {
			pid, ok := pathID(c)
			if !ok {
				return
			}
			err := svc.Add(c.Request.Context(), httpx.RequireUserID(c), pid)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		fav.DELETE("/:productId", func(c *gin.Context) {
			pid, ok := pathID(c)
			if !ok {
				return
			}
			err := svc.Remove(c.Request.Context(), httpx.RequireUserID(c), pid)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})
	}
}

func dispatch(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var be httpx.BizError
	if errors.As(err, &be) {
		httpx.Abort(c, be)
		return true
	}
	panic(err)
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		httpx.Abort(c, httpx.Biz("参数错误"))
		return 0, false
	}
	return id, true
}
```

- [ ] **Step 2: Build check**

```bash
cd server && go build ./internal/favorite/...
```

Expected: No errors.

---

## Task 4: Write favorite service tests

**Files:**
- Create: `server/internal/favorite/service_test.go`

- [ ] **Step 1: Write service_test.go**

```go
package favorite

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

type stubQuerier struct {
	products   map[int64]gen.Product
	favorites  map[int64]map[int64]bool // userID -> productID -> exists
	productIDs []int64                  // ordered result for ListFavoriteProductIDs
	images     []gen.ProductImage
	users      []gen.ListUsersByIDsRow
	cats       []gen.ListCategoriesByIDsRow

	insertedFav gen.InsertFavoriteParams
	deletedFav  gen.DeleteFavoriteParams
}

func (s *stubQuerier) InsertFavorite(_ context.Context, arg gen.InsertFavoriteParams) error {
	s.insertedFav = arg
	if s.favorites == nil {
		s.favorites = map[int64]map[int64]bool{}
	}
	if s.favorites[arg.UserID] == nil {
		s.favorites[arg.UserID] = map[int64]bool{}
	}
	s.favorites[arg.UserID][arg.ProductID] = true
	return nil
}
func (s *stubQuerier) DeleteFavorite(_ context.Context, arg gen.DeleteFavoriteParams) error {
	s.deletedFav = arg
	return nil
}
func (s *stubQuerier) CountFavorite(_ context.Context, arg gen.CountFavoriteParams) (int64, error) {
	if s.favorites != nil && s.favorites[arg.UserID] != nil && s.favorites[arg.UserID][arg.ProductID] {
		return 1, nil
	}
	return 0, nil
}
func (s *stubQuerier) ListFavoriteProductIDs(_ context.Context, _ int64) ([]int64, error) {
	return s.productIDs, nil
}
func (s *stubQuerier) GetProduct(_ context.Context, id int64) (gen.Product, error) {
	p, ok := s.products[id]
	if !ok {
		return gen.Product{}, pgx.ErrNoRows
	}
	return p, nil
}
func (s *stubQuerier) ListProductsByIDs(_ context.Context, _ []int64) ([]gen.Product, error) {
	out := make([]gen.Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	return out, nil
}
func (s *stubQuerier) ListProductImages(_ context.Context, _ []int64) ([]gen.ProductImage, error) {
	return s.images, nil
}
func (s *stubQuerier) ListUsersByIDs(_ context.Context, _ []int64) ([]gen.ListUsersByIDsRow, error) {
	return s.users, nil
}
func (s *stubQuerier) ListCategoriesByIDs(_ context.Context, _ []int64) ([]gen.ListCategoriesByIDsRow, error) {
	return s.cats, nil
}

func makeProduct(id int64) gen.Product {
	var n pgtype.Numeric
	_ = n.Scan("10.00")
	return gen.Product{ID: id, SellerID: 1, CategoryID: 2, Title: "Test", Price: n, OriginalPrice: n, Status: "ON_SALE"}
}

// --- Tests ---

func TestAddProductNotFound(t *testing.T) {
	svc := NewService(&stubQuerier{})
	err := svc.Add(context.Background(), 1, 999)
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "商品不存在" {
		t.Fatalf("unexpected msg: %q", be.Msg)
	}
}

func TestAddIdempotent(t *testing.T) {
	stub := &stubQuerier{
		products:  map[int64]gen.Product{1: makeProduct(1)},
		favorites: map[int64]map[int64]bool{10: {1: true}},
	}
	svc := NewService(stub)
	// Adding again should not call InsertFavorite (insertedFav stays zero value)
	if err := svc.Add(context.Background(), 10, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.insertedFav.ProductID != 0 {
		t.Fatal("expected InsertFavorite NOT to be called for duplicate")
	}
}

func TestAddNewFavorite(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{1: makeProduct(1)},
	}
	svc := NewService(stub)
	if err := svc.Add(context.Background(), 10, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.insertedFav.UserID != 10 || stub.insertedFav.ProductID != 1 {
		t.Fatalf("InsertFavorite called with wrong params: %+v", stub.insertedFav)
	}
}

func TestRemoveCallsDelete(t *testing.T) {
	stub := &stubQuerier{}
	svc := NewService(stub)
	if err := svc.Remove(context.Background(), 10, 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.deletedFav.UserID != 10 || stub.deletedFav.ProductID != 5 {
		t.Fatalf("DeleteFavorite called with wrong params: %+v", stub.deletedFav)
	}
}

func TestMyFavoritesEmpty(t *testing.T) {
	stub := &stubQuerier{productIDs: []int64{}}
	svc := NewService(stub)
	page, err := svc.MyFavorites(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected total 0, got %d", page.Total)
	}
}

func TestMyFavoritesReturnsFavoritedTrue(t *testing.T) {
	stub := &stubQuerier{
		productIDs: []int64{1},
		products:   map[int64]gen.Product{1: makeProduct(1)},
	}
	svc := NewService(stub)
	page, err := svc.MyFavorites(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected total 1, got %d", page.Total)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd server && go test ./internal/favorite/... -v
```

Expected: All tests PASS.

---

## Task 5: Write upload service

**Files:**
- Create: `server/internal/upload/service.go`

- [ ] **Step 1: Write service.go**

```go
package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

var allowedExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true,
}

// Service handles image uploads.
type Service struct {
	uploadDir   string
	urlPrefix   string
	maxFileSize int64
}

// NewService constructs an upload service.
// uploadDir: local directory to store files (e.g. "./uploads")
// urlPrefix: URL prefix for returned URLs (e.g. "/uploads")
// maxFileSize: max allowed bytes (e.g. 5242880)
func NewService(uploadDir, urlPrefix string, maxFileSize int64) *Service {
	return &Service{
		uploadDir:   uploadDir,
		urlPrefix:   urlPrefix,
		maxFileSize: maxFileSize,
	}
}

// Upload validates and saves the uploaded image file.
// Returns the public URL path on success.
// Matches Java UploadService.upload() exactly.
func (s *Service) Upload(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Size == 0 {
		return "", httpx.Biz("文件不能为空")
	}
	if file.Size > s.maxFileSize {
		return "", httpx.Biz("文件不能超过 5MB")
	}

	origin := file.Filename
	if origin == "" || !strings.Contains(origin, ".") {
		return "", httpx.Biz("文件名非法")
	}
	ext := strings.ToLower(origin[strings.LastIndex(origin, ".")+1:])
	if !allowedExts[ext] {
		return "", httpx.Biz("仅支持 jpg/jpeg/png/gif/webp")
	}

	// Dated subdirectory: yyyy-MM-dd
	dateDir := time.Now().Format("2006-01-02")
	dir := filepath.Join(s.uploadDir, dateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", httpx.Biz("创建目录失败")
	}

	// UUID-like hex filename (matches Java UUID.randomUUID().toString().replace("-",""))
	filename := randomHex(16) + "." + ext
	dest := filepath.Join(dir, filename)

	if err := saveFile(file, dest); err != nil {
		return "", httpx.Biz("保存文件失败")
	}

	url := s.urlPrefix + "/" + dateDir + "/" + filename
	return url, nil
}

// randomHex generates n random bytes and returns them as a 2n-char hex string.
// 16 bytes → 32-char hex, matching Java's UUID hex (32 chars).
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// fallback: use fmt with current time nanos (should never happen)
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func saveFile(fh *multipart.FileHeader, dest string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}
```

- [ ] **Step 2: Build check**

```bash
cd server && go build ./internal/upload/...
```

Expected: No errors.

---

## Task 6: Write upload handler

**Files:**
- Create: `server/internal/upload/handler.go`

- [ ] **Step 1: Write handler.go**

```go
package upload

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the upload route onto the /api group.
// Requires authentication.
//
//	POST /api/upload/image   upload an image file (multipart field: "file")
func Register(api *gin.RouterGroup, svc *Service) {
	ul := api.Group("/upload")
	ul.Use(httpx.RequireAuth())
	{
		ul.POST("/image", func(c *gin.Context) {
			file, err := c.FormFile("file")
			if err != nil || file == nil {
				httpx.Abort(c, httpx.Biz("文件不能为空"))
				return
			}
			url, err := svc.Upload(file)
			var be httpx.BizError
			if errors.As(err, &be) {
				httpx.Abort(c, be)
				return
			}
			if err != nil {
				panic(err)
			}
			c.JSON(200, httpx.OK(map[string]string{"url": url}))
		})
	}
}
```

- [ ] **Step 2: Build check**

```bash
cd server && go build ./internal/upload/...
```

Expected: No errors.

---

## Task 7: Write upload service tests

**Files:**
- Create: `server/internal/upload/service_test.go`

- [ ] **Step 1: Write service_test.go**

```go
package upload

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// makeFileHeader creates a *multipart.FileHeader with the given filename and content.
// This avoids needing an actual HTTP request in unit tests.
func makeFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("createFormFile: %v", err)
	}
	if _, err := io.Copy(fw, bytes.NewReader(content)); err != nil {
		t.Fatalf("copy: %v", err)
	}
	w.Close()

	r := multipart.NewReader(&buf, w.Boundary())
	form, err := r.ReadForm(10 << 20)
	if err != nil {
		t.Fatalf("readForm: %v", err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("no file in form")
	}
	return files[0]
}

func bizMsg(err error) string {
	var be httpx.BizError
	if errors.As(err, &be) {
		return be.Msg
	}
	return ""
}

func TestUploadEmptyFile(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 5<<20)
	fh := makeFileHeader(t, "test.jpg", []byte{})
	_, err := svc.Upload(fh)
	if bizMsg(err) != "文件不能为空" {
		t.Fatalf("expected 文件不能为空, got %v", err)
	}
}

func TestUploadTooLarge(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 10) // max 10 bytes
	fh := makeFileHeader(t, "test.jpg", bytes.Repeat([]byte("x"), 20))
	_, err := svc.Upload(fh)
	if bizMsg(err) != "文件不能超过 5MB" {
		t.Fatalf("expected 文件不能超过 5MB, got %v", err)
	}
}

func TestUploadInvalidExtension(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 5<<20)
	fh := makeFileHeader(t, "test.exe", []byte("data"))
	_, err := svc.Upload(fh)
	if bizMsg(err) != "仅支持 jpg/jpeg/png/gif/webp" {
		t.Fatalf("expected extension error, got %v", err)
	}
}

func TestUploadNoExtension(t *testing.T) {
	svc := NewService(t.TempDir(), "/uploads", 5<<20)
	fh := makeFileHeader(t, "testfile", []byte("data"))
	_, err := svc.Upload(fh)
	if bizMsg(err) != "文件名非法" {
		t.Fatalf("expected 文件名非法, got %v", err)
	}
}

func TestUploadSuccessPathScheme(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	fh := makeFileHeader(t, "photo.jpg", []byte{0xff, 0xd8, 0xff}) // fake jpeg bytes
	url, err := svc.Upload(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// URL must start with /uploads/<today>/
	today := time.Now().Format("2006-01-02")
	prefix := "/uploads/" + today + "/"
	if !strings.HasPrefix(url, prefix) {
		t.Fatalf("URL %q does not start with %q", url, prefix)
	}
	// URL must end with .jpg
	if !strings.HasSuffix(url, ".jpg") {
		t.Fatalf("URL %q does not end with .jpg", url)
	}

	// Verify file exists on disk
	// URL is /uploads/2026-01-02/abc123.jpg; file is at dir/2026-01-02/abc123.jpg
	filename := filepath.Base(url)
	diskPath := filepath.Join(dir, today, filename)
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatalf("file not found on disk at %q", diskPath)
	}
}

func TestUploadHexFilenameLength(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	fh := makeFileHeader(t, "img.png", []byte{0x89, 0x50, 0x4e, 0x47})
	url, err := svc.Upload(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// filename part (without extension) should be 32 hex chars
	base := strings.TrimSuffix(filepath.Base(url), ".png")
	if len(base) != 32 {
		t.Fatalf("expected 32-char hex filename, got %d chars: %q", len(base), base)
	}
}

func TestAllowedExtensions(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir, "/uploads", 5<<20)
	for _, ext := range []string{"jpg", "jpeg", "png", "gif", "webp"} {
		fh := makeFileHeader(t, "file."+ext, []byte("data"))
		if _, err := svc.Upload(fh); err != nil {
			t.Fatalf("extension %q should be allowed, got error: %v", ext, err)
		}
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd server && go test ./internal/upload/... -v
```

Expected: All tests PASS.

---

## Task 8: Register routes in main.go

**Files:**
- Modify: `server/cmd/smudeal/main.go`

- [ ] **Step 1: Add imports and register both domains**

Add to imports:
```go
"github.com/John-DengD/smu-deal/server/internal/favorite"
"github.com/John-DengD/smu-deal/server/internal/upload"
```

Add after the existing `announcement.Register(...)` line:
```go
favorite.Register(api, favorite.NewService(q))
upload.Register(api, upload.NewService(cfg.UploadDir, cfg.URLPrefix, cfg.MaxFileSize))
```

- [ ] **Step 2: Build and test the whole server**

```bash
cd server && go build ./... && go vet ./... && go test ./...
```

Expected: All pass, no errors.

---

## Task 9: End-to-end verification

- [ ] **Step 1: Start server with real DB**

```bash
cd server && DB_URL="postgres://smu_deal:password@localhost:5432/smu_deal?sslmode=disable" go run ./cmd/smudeal
```

- [ ] **Step 2: Login and get token**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"studentNo":"student001","password":"123456"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
echo "TOKEN=$TOKEN"
```

- [ ] **Step 3: Create a product to favorite**

```bash
PRODUCT_ID=$(curl -s -X POST http://localhost:8080/api/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Product","categoryId":1,"price":"9.99","images":[]}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
echo "PRODUCT_ID=$PRODUCT_ID"
```

- [ ] **Step 4: Test favorite add (idempotent)**

```bash
# First add
curl -s -X POST http://localhost:8080/api/favorites/$PRODUCT_ID \
  -H "Authorization: Bearer $TOKEN"
# Second add (must still return code:0, not error)
curl -s -X POST http://localhost:8080/api/favorites/$PRODUCT_ID \
  -H "Authorization: Bearer $TOKEN"
```

Expected both: `{"code":0,"message":"ok","data":null}`

- [ ] **Step 5: Test GET /api/favorites**

```bash
curl -s http://localhost:8080/api/favorites \
  -H "Authorization: Bearer $TOKEN"
```

Expected: `{"code":0,...,"data":{"total":1,"records":[{"id":...,"title":"Test Product",...,"favorited":true,...}]}}`

- [ ] **Step 6: Test DELETE and verify removal**

```bash
curl -s -X DELETE http://localhost:8080/api/favorites/$PRODUCT_ID \
  -H "Authorization: Bearer $TOKEN"
curl -s http://localhost:8080/api/favorites \
  -H "Authorization: Bearer $TOKEN"
```

Expected: delete returns `{"code":0,...,"data":null}`. GET returns `{"total":0,"records":[]}`.

- [ ] **Step 7: Test upload with a real image**

```bash
# Create a tiny valid PNG (1x1 pixel)
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82' > /tmp/test.png

UPLOAD_RESP=$(curl -s -X POST http://localhost:8080/api/upload/image \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/test.png")
echo $UPLOAD_RESP
```

Expected: `{"code":0,"message":"ok","data":{"url":"/uploads/2026-07-23/...32hex....png"}}`

- [ ] **Step 8: Verify static serving**

```bash
URL=$(echo $UPLOAD_RESP | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['url'])")
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080$URL
```

Expected: `200`

- [ ] **Step 9: Test 401 without token**

```bash
curl -s -X POST http://localhost:8080/api/upload/image \
  -F "file=@/tmp/test.png"
```

Expected: HTTP 401, `{"code":401,...}`

- [ ] **Step 10: Test oversized file**

```bash
# 6MB file
dd if=/dev/zero bs=1024 count=6144 2>/dev/null | \
  curl -s -X POST http://localhost:8080/api/upload/image \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@-;filename=big.jpg"
```

Expected: `{"code":400,"message":"文件不能超过 5MB",...}`

- [ ] **Step 11: Test invalid extension**

```bash
printf 'data' > /tmp/test.exe
curl -s -X POST http://localhost:8080/api/upload/image \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/test.exe"
```

Expected: `{"code":400,"message":"仅支持 jpg/jpeg/png/gif/webp",...}`

- [ ] **Step 12: Cleanup**

```bash
# Delete test product (sets OFFLINE)
curl -s -X DELETE http://localhost:8080/api/products/$PRODUCT_ID \
  -H "Authorization: Bearer $TOKEN"
# Delete uploaded file
URL_PATH=$(echo $UPLOAD_RESP | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['url'])")
rm -f "./uploads${URL_PATH#/uploads}"
rm -f /tmp/test.png /tmp/test.exe
```

---

## Task 10: Commit

- [ ] **Step 1: Commit**

```bash
cd /Users/new/sch-projects/smu-deal
git add server/internal/db/queries/favorite.sql \
        server/internal/db/gen/favorite.sql.go \
        server/internal/favorite/handler.go \
        server/internal/favorite/service.go \
        server/internal/favorite/service_test.go \
        server/internal/upload/handler.go \
        server/internal/upload/service.go \
        server/internal/upload/service_test.go \
        server/cmd/smudeal/main.go
git commit -m "feat(server): add favorite and upload domains (Tasks 11 & 19)"
```

---

## Self-Review Notes

1. **Spec coverage:**
   - Favorite: GET list (product items with `favorited:true`) ✓, POST add (idempotent) ✓, DELETE remove ✓, all auth-required ✓
   - Upload: POST /api/upload/image, field name `file` ✓, extensions `jpg/jpeg/png/gif/webp` ✓, max size from cfg ✓, dated subdir ✓, 32-char hex UUID ✓, response `{"url":"..."}` ✓

2. **product.Price field name:** The plan references `product.Price{Numeric: p.Price}`. Task 2 Step 2 checks the actual field name in `price.go` before writing service.go.

3. **ListProductsByIDs:** Task 2 Step 3 checks if this query exists already. If not, adds it to favorite.sql.

4. **`time` import in service.go:** The `timePtr` function uses `time.Time` — ensure `"time"` is imported in favorite/service.go.

5. **`product.Item.Price` is embedded or named?** The service.go in product package has `Price Price` — it's a named field of type `Price` (not embedded). `product.Price` has `pgtype.Numeric` embedded: check `price.go` for exact field name.
