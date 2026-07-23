package favorite

import (
	"context"
	"errors"
	"time"

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
	// product enrichment queries (shared with product domain)
	GetProduct(ctx context.Context, id int64) (gen.Product, error)
	ListProductsByIDs(ctx context.Context, ids []int64) ([]gen.Product, error)
	ListProductImages(ctx context.Context, productIds []int64) ([]gen.ProductImage, error)
	ListUsersByIDs(ctx context.Context, ids []int64) ([]gen.ListUsersByIDsRow, error)
	ListCategoriesByIDs(ctx context.Context, ids []int64) ([]gen.ListCategoriesByIDsRow, error)
}

// Service handles favorites business logic.
type Service struct {
	q Querier
}

// NewService constructs the favorite service.
func NewService(q Querier) *Service {
	return &Service{q: q}
}

// Add adds a product to the user's favorites.
// Matches Java FavoriteService.add:
//   - returns 商品不存在 if product not found
//   - idempotent: silently returns ok if already favorited
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

// Remove removes a product from the user's favorites.
// Matches Java FavoriteService.remove: always succeeds (no error if not favorited).
func (s *Service) Remove(ctx context.Context, userID, productID int64) error {
	return s.q.DeleteFavorite(ctx, gen.DeleteFavoriteParams{UserID: userID, ProductID: productID})
}

// MyFavorites returns the current user's favorited products as enriched product items.
// Matches Java FavoriteService.myFavorites: returns PageResult<ProductDTO.Item>
// with the full product shape and favorited=true on every item.
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

	// Assemble items in favorite order (productIDs is already ordered by created_at DESC).
	// Matches Java: iterates favorites list, skips products that have been deleted.
	items := make([]product.Item, 0, len(productIDs))
	for _, pid := range productIDs {
		p, ok := pmap[pid]
		if !ok {
			continue // product deleted since favorited
		}
		it := product.Item{
			ID:             p.ID,
			Title:          p.Title,
			Description:    nil, // Java FavoriteService.myFavorites does not set description (stays null)
			Price:          product.Price{Numeric: p.Price},
			OriginalPrice:  product.Price{Numeric: p.OriginalPrice},
			ConditionLevel: p.ConditionLevel,
			TradeLocation:  p.TradeLocation,
			Status:         p.Status,
			ViewCount:      p.ViewCount,
			CreatedAt:      productCreatedAt(p),
			CategoryID:     p.CategoryID,
			SellerID:       p.SellerID,
			Favorited:      true, // always true in the favorites list
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

func productCreatedAt(p gen.Product) *time.Time {
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
