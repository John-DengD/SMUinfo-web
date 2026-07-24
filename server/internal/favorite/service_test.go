package favorite

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
	"github.com/John-DengD/smu-deal/server/internal/product"
)

// stubQuerier implements Querier for unit tests without a real DB.
type stubQuerier struct {
	products   map[int64]gen.Product
	favorites  map[int64]map[int64]bool // userID -> productID -> exists
	productIDs []int64                  // ordered result for ListFavoriteProductIDs
	images     []gen.ProductImage
	users      []gen.ListUsersByIDsRow
	cats       []gen.ListCategoriesByIDsRow

	insertedFav gen.InsertFavoriteParams
	insertCalled bool
	deletedFav  gen.DeleteFavoriteParams
}

func (s *stubQuerier) InsertFavorite(_ context.Context, arg gen.InsertFavoriteParams) error {
	s.insertedFav = arg
	s.insertCalled = true
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

// makeProduct creates a minimal gen.Product for testing.
func makeProduct(id int64) gen.Product {
	var n pgtype.Numeric
	_ = n.Scan("10.00")
	return gen.Product{
		ID: id, SellerID: 1, CategoryID: 2,
		Title: "Test Product", Price: n, OriginalPrice: n, Status: "ON_SALE",
	}
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
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestAddIdempotentDoesNotInsert(t *testing.T) {
	stub := &stubQuerier{
		products:  map[int64]gen.Product{1: makeProduct(1)},
		favorites: map[int64]map[int64]bool{10: {1: true}},
	}
	svc := NewService(stub)
	// Adding a duplicate should succeed silently and NOT call InsertFavorite
	if err := svc.Add(context.Background(), 10, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.insertCalled {
		t.Fatal("expected InsertFavorite NOT to be called for duplicate")
	}
}

func TestAddNewFavoriteInsertsRow(t *testing.T) {
	stub := &stubQuerier{
		products: map[int64]gen.Product{1: makeProduct(1)},
	}
	svc := NewService(stub)
	if err := svc.Add(context.Background(), 10, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.insertCalled {
		t.Fatal("expected InsertFavorite to be called")
	}
	if stub.insertedFav.UserID != 10 || stub.insertedFav.ProductID != 1 {
		t.Fatalf("InsertFavorite called with wrong params: %+v", stub.insertedFav)
	}
}

func TestRemoveCallsDeleteWithCorrectParams(t *testing.T) {
	stub := &stubQuerier{}
	svc := NewService(stub)
	if err := svc.Remove(context.Background(), 10, 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.deletedFav.UserID != 10 || stub.deletedFav.ProductID != 5 {
		t.Fatalf("DeleteFavorite called with wrong params: %+v", stub.deletedFav)
	}
}

func TestMyFavoritesEmptyListReturnsTotalZero(t *testing.T) {
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
	records, ok := page.Records.([]product.Item)
	if !ok {
		t.Fatalf("expected Records to be []product.Item, got %T", page.Records)
	}
	if len(records) == 0 {
		t.Fatal("expected at least one record")
	}
	if !records[0].Favorited {
		t.Fatalf("expected Favorited == true, got false")
	}
}

func TestMyFavoritesSkipsDeletedProducts(t *testing.T) {
	// productIDs includes ID 99 which is not in products map (simulates deletion)
	stub := &stubQuerier{
		productIDs: []int64{1, 99},
		products:   map[int64]gen.Product{1: makeProduct(1)},
	}
	svc := NewService(stub)
	page, err := svc.MyFavorites(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Total reflects actual assembled items (1), not raw favorite rows (2)
	if page.Total != 1 {
		t.Fatalf("expected total 1 (skipping deleted product), got %d", page.Total)
	}
}
