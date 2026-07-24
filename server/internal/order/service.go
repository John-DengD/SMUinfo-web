package order

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
	"github.com/John-DengD/smu-deal/server/internal/product"
)

// TxBeginner begins a database transaction. Satisfied by *pgxpool.Pool.
// The state-transition writes (confirm/finish/cancel) mutate order status,
// product status and sibling orders atomically, matching Java @Transactional.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Querier is the subset of the sqlc-generated *gen.Queries the order service
// needs for reads and the single-write create path.
type Querier interface {
	GetProduct(ctx context.Context, id int64) (gen.Product, error)
	GetOrder(ctx context.Context, id int64) (gen.TradeOrder, error)
	InsertOrder(ctx context.Context, arg gen.InsertOrderParams) (gen.TradeOrder, error)
	ListOrdersByBuyer(ctx context.Context, buyerID int64) ([]gen.TradeOrder, error)
	ListOrdersBySeller(ctx context.Context, sellerID int64) ([]gen.TradeOrder, error)
	ListOrdersByUser(ctx context.Context, userID int64) ([]gen.TradeOrder, error)
	CountActiveOrdersByBuyerProduct(ctx context.Context, arg gen.CountActiveOrdersByBuyerProductParams) (int64, error)
	CountReservedOrdersOtherThan(ctx context.Context, arg gen.CountReservedOrdersOtherThanParams) (int64, error)
	CountActiveOrdersOtherThan(ctx context.Context, arg gen.CountActiveOrdersOtherThanParams) (int64, error)
	ListPendingOrdersOtherThan(ctx context.Context, arg gen.ListPendingOrdersOtherThanParams) ([]gen.TradeOrder, error)
	ListActiveOrdersOtherThan(ctx context.Context, arg gen.ListActiveOrdersOtherThanParams) ([]gen.TradeOrder, error)
	// enrich support (reused from product domain queries).
	ListProductImages(ctx context.Context, productIds []int64) ([]gen.ProductImage, error)
	ListUsersByIDs(ctx context.Context, ids []int64) ([]gen.ListUsersByIDsRow, error)
}

// txQuerier is the set of write operations the transactional transition paths
// run. Both *gen.Queries (production) and the test stub satisfy it.
type txQuerier interface {
	SetOrderStatus(ctx context.Context, arg gen.SetOrderStatusParams) error
	SetOrderCompleted(ctx context.Context, id int64) error
	SetProductStatus(ctx context.Context, arg gen.SetProductStatusParams) error
	ListPendingOrdersOtherThan(ctx context.Context, arg gen.ListPendingOrdersOtherThanParams) ([]gen.TradeOrder, error)
	ListActiveOrdersOtherThan(ctx context.Context, arg gen.ListActiveOrdersOtherThanParams) ([]gen.TradeOrder, error)
	CountReservedOrdersOtherThan(ctx context.Context, arg gen.CountReservedOrdersOtherThanParams) (int64, error)
	CountActiveOrdersOtherThan(ctx context.Context, arg gen.CountActiveOrdersOtherThanParams) (int64, error)
	InsertProductComment(ctx context.Context, arg gen.InsertProductCommentParams) (gen.ProductComment, error)
	GetProduct(ctx context.Context, id int64) (gen.Product, error)
}

type Service struct {
	q     Querier
	pool  TxBeginner
	newTx func(tx pgx.Tx) txQuerier
}

// NewService constructs the order service. pool is used by the state-transition
// paths to run the multi-row updates atomically. In production both q and the
// tx writer come from the same *pgxpool.Pool.
func NewService(q Querier, pool TxBeginner) *Service {
	return &Service{
		q:     q,
		pool:  pool,
		newTx: func(tx pgx.Tx) txQuerier { return gen.New(tx) },
	}
}

// Item mirrors OrderDTO.Item (camelCase wire contract). productPrice reuses the
// product.Price type for fixed-scale-2 fidelity, matching Java BigDecimal.
type Item struct {
	ID           int64         `json:"id"`
	ProductID    int64         `json:"productId"`
	ProductTitle *string       `json:"productTitle"`
	ProductCover *string       `json:"productCover"`
	ProductPrice product.Price `json:"productPrice"`
	BuyerID      int64         `json:"buyerId"`
	BuyerName    *string       `json:"buyerName"`
	SellerID     int64         `json:"sellerId"`
	SellerName   *string       `json:"sellerName"`
	Status       string        `json:"status"`
	MeetLocation *string       `json:"meetLocation"`
	Remark       *string       `json:"remark"`
	CreatedAt    *time.Time    `json:"createdAt"`
	UpdatedAt    *time.Time    `json:"updatedAt"`
	CompletedAt  *time.Time    `json:"completedAt"`
}

// CreateReq mirrors OrderDTO.CreateReq.
type CreateReq struct {
	ProductID    *int64  `json:"productId"`
	MeetLocation *string `json:"meetLocation"`
	Remark       *string `json:"remark"`
}

// Create replicates OrderService.create: buyer opens a trade request.
func (s *Service) Create(ctx context.Context, buyerID int64, req CreateReq) (Item, error) {
	if req.ProductID == nil {
		return Item{}, httpx.Biz("参数错误")
	}
	p, err := s.q.GetProduct(ctx, *req.ProductID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, httpx.Biz("商品不存在")
		}
		return Item{}, err
	}
	if p.SellerID == buyerID {
		return Item{}, httpx.Biz("不能购买自己的商品")
	}
	if p.Status != "ON_SALE" {
		return Item{}, httpx.Biz("商品当前不可购买")
	}
	active, err := s.q.CountActiveOrdersByBuyerProduct(ctx, gen.CountActiveOrdersByBuyerProductParams{
		ProductID: *req.ProductID,
		BuyerID:   buyerID,
	})
	if err != nil {
		return Item{}, err
	}
	if active > 0 {
		return Item{}, httpx.Biz("你已经提交过预约申请")
	}

	o, err := s.q.InsertOrder(ctx, gen.InsertOrderParams{
		ProductID:    *req.ProductID,
		BuyerID:      buyerID,
		SellerID:     p.SellerID,
		Status:       "PENDING",
		MeetLocation: req.MeetLocation,
		Remark:       req.Remark,
	})
	if err != nil {
		return Item{}, err
	}
	items, err := s.enrich(ctx, []gen.TradeOrder{o})
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

// MyOrders replicates OrderService.myOrders: filter by role, order by createdAt desc.
func (s *Service) MyOrders(ctx context.Context, userID int64, role string) ([]Item, error) {
	var orders []gen.TradeOrder
	var err error
	switch strings.ToLower(role) {
	case "seller":
		orders, err = s.q.ListOrdersBySeller(ctx, userID)
	case "buyer":
		orders, err = s.q.ListOrdersByBuyer(ctx, userID)
	default:
		orders, err = s.q.ListOrdersByUser(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	return s.enrich(ctx, orders)
}

// Confirm replicates OrderService.confirm: seller-only; PENDING -> RESERVED,
// flips product to RESERVED, cancels sibling PENDING orders, adds a comment.
func (s *Service) Confirm(ctx context.Context, orderID, userID int64) (Item, error) {
	o, err := s.getAndCheck(ctx, orderID, userID, true)
	if err != nil {
		return Item{}, err
	}
	if o.Status != "PENDING" {
		return Item{}, httpx.Biz("当前状态不能确认预约")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Item{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.newTx(tx)

	reserved, err := qtx.CountReservedOrdersOtherThan(ctx, gen.CountReservedOrdersOtherThanParams{
		ProductID: o.ProductID,
		ID:        o.ID,
	})
	if err != nil {
		return Item{}, err
	}
	if reserved > 0 {
		return Item{}, httpx.Biz("该商品已有确认的预约")
	}

	if err := qtx.SetOrderStatus(ctx, gen.SetOrderStatusParams{ID: o.ID, Status: "RESERVED"}); err != nil {
		return Item{}, err
	}
	o.Status = "RESERVED"

	if err := qtx.SetProductStatus(ctx, gen.SetProductStatusParams{ID: o.ProductID, Status: "RESERVED"}); err != nil {
		return Item{}, err
	}

	pending, err := qtx.ListPendingOrdersOtherThan(ctx, gen.ListPendingOrdersOtherThanParams{
		ProductID: o.ProductID,
		ID:        o.ID,
	})
	if err != nil {
		return Item{}, err
	}
	for _, other := range pending {
		if err := qtx.SetOrderStatus(ctx, gen.SetOrderStatusParams{ID: other.ID, Status: "CANCELLED"}); err != nil {
			return Item{}, err
		}
	}

	if err := addReservationComment(ctx, qtx, s.q, o); err != nil {
		return Item{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Item{}, err
	}
	items, err := s.enrich(ctx, []gen.TradeOrder{o})
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

// Finish replicates OrderService.finish: buyer or seller; RESERVED -> COMPLETED,
// sets completedAt, flips product to SOLD, cancels sibling active orders.
func (s *Service) Finish(ctx context.Context, orderID, userID int64) (Item, error) {
	o, err := s.getAndCheck(ctx, orderID, userID, false)
	if err != nil {
		return Item{}, err
	}
	if o.Status != "RESERVED" {
		return Item{}, httpx.Biz("请先由卖家确认预约")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Item{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.newTx(tx)

	if err := qtx.SetOrderCompleted(ctx, o.ID); err != nil {
		return Item{}, err
	}
	if err := qtx.SetProductStatus(ctx, gen.SetProductStatusParams{ID: o.ProductID, Status: "SOLD"}); err != nil {
		return Item{}, err
	}

	others, err := qtx.ListActiveOrdersOtherThan(ctx, gen.ListActiveOrdersOtherThanParams{
		ProductID: o.ProductID,
		ID:        o.ID,
	})
	if err != nil {
		return Item{}, err
	}
	for _, other := range others {
		if err := qtx.SetOrderStatus(ctx, gen.SetOrderStatusParams{ID: other.ID, Status: "CANCELLED"}); err != nil {
			return Item{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Item{}, err
	}
	// Re-read to reflect the trigger-updated completed_at/updated_at.
	fresh, err := s.q.GetOrder(ctx, o.ID)
	if err != nil {
		return Item{}, err
	}
	items, err := s.enrich(ctx, []gen.TradeOrder{fresh})
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

// Cancel replicates OrderService.cancel: buyer or seller; not allowed once
// COMPLETED; -> CANCELLED; reverts product to ON_SALE if it was RESERVED and no
// other active orders remain.
func (s *Service) Cancel(ctx context.Context, orderID, userID int64) (Item, error) {
	o, err := s.getAndCheck(ctx, orderID, userID, false)
	if err != nil {
		return Item{}, err
	}
	if o.Status == "COMPLETED" {
		return Item{}, httpx.Biz("已完成订单不可取消")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Item{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.newTx(tx)

	if err := qtx.SetOrderStatus(ctx, gen.SetOrderStatusParams{ID: o.ID, Status: "CANCELLED"}); err != nil {
		return Item{}, err
	}
	o.Status = "CANCELLED"

	active, err := qtx.CountActiveOrdersOtherThan(ctx, gen.CountActiveOrdersOtherThanParams{
		ProductID: o.ProductID,
		ID:        o.ID,
	})
	if err != nil {
		return Item{}, err
	}
	if active == 0 {
		p, err := qtx.GetProduct(ctx, o.ProductID)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return Item{}, err
			}
		} else if p.Status == "RESERVED" {
			if err := qtx.SetProductStatus(ctx, gen.SetProductStatusParams{ID: o.ProductID, Status: "ON_SALE"}); err != nil {
				return Item{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Item{}, err
	}
	items, err := s.enrich(ctx, []gen.TradeOrder{o})
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

// getAndCheck loads the order and enforces the role guard, matching Java.
func (s *Service) getAndCheck(ctx context.Context, orderID, userID int64, sellerOnly bool) (gen.TradeOrder, error) {
	o, err := s.q.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.TradeOrder{}, httpx.Biz("订单不存在")
		}
		return gen.TradeOrder{}, err
	}
	if sellerOnly {
		if o.SellerID != userID {
			return gen.TradeOrder{}, httpx.NewBiz(403, "无权操作")
		}
	} else {
		if o.SellerID != userID && o.BuyerID != userID {
			return gen.TradeOrder{}, httpx.NewBiz(403, "无权操作")
		}
	}
	return o, nil
}

// addReservationComment mirrors OrderService.addReservationComment: posts a
// system comment on the product that the buyer reserved it. The buyer name is
// looked up via the (non-tx) read querier; the write goes through the tx.
func addReservationComment(ctx context.Context, qtx txQuerier, q Querier, o gen.TradeOrder) error {
	buyerName := "同学"
	users, err := q.ListUsersByIDs(ctx, []int64{o.BuyerID})
	if err != nil {
		return err
	}
	if len(users) > 0 {
		buyerName = users[0].Name
	}
	content := html.EscapeString(buyerName + " 已预约成功了这件商品")
	_, err = qtx.InsertProductComment(ctx, gen.InsertProductCommentParams{
		ProductID: o.ProductID,
		UserID:    o.BuyerID,
		Content:   content,
	})
	return err
}

// enrich replicates OrderService.enrich: batch-load buyer/seller names, product
// title/price and cover image.
func (s *Service) enrich(ctx context.Context, orders []gen.TradeOrder) ([]Item, error) {
	if len(orders) == 0 {
		return make([]Item, 0), nil
	}

	userSet := map[int64]struct{}{}
	productSet := map[int64]struct{}{}
	for _, o := range orders {
		userSet[o.BuyerID] = struct{}{}
		userSet[o.SellerID] = struct{}{}
		productSet[o.ProductID] = struct{}{}
	}

	users, err := s.q.ListUsersByIDs(ctx, keys(userSet))
	if err != nil {
		return nil, err
	}
	userMap := map[int64]gen.ListUsersByIDsRow{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	productIDs := keys(productSet)
	// Products are loaded individually via GetProduct to reuse the existing
	// query; there are at most a handful per response.
	productMap := map[int64]gen.Product{}
	for _, pid := range productIDs {
		p, err := s.q.GetProduct(ctx, pid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		productMap[pid] = p
	}

	imgs, err := s.q.ListProductImages(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	// First image per product (sorted by sort_order asc) is the cover.
	coverMap := map[int64]string{}
	for _, img := range imgs {
		if _, ok := coverMap[img.ProductID]; !ok {
			coverMap[img.ProductID] = img.ImageUrl
		}
	}

	result := make([]Item, 0, len(orders))
	for _, o := range orders {
		it := Item{
			ID:           o.ID,
			ProductID:    o.ProductID,
			BuyerID:      o.BuyerID,
			SellerID:     o.SellerID,
			Status:       o.Status,
			MeetLocation: o.MeetLocation,
			Remark:       o.Remark,
			CreatedAt:    timePtr(o.CreatedAt),
			UpdatedAt:    timePtr(o.UpdatedAt),
			CompletedAt:  timePtr(o.CompletedAt),
		}
		if b, ok := userMap[o.BuyerID]; ok {
			n := b.Name
			it.BuyerName = &n
		}
		if se, ok := userMap[o.SellerID]; ok {
			n := se.Name
			it.SellerName = &n
		}
		if p, ok := productMap[o.ProductID]; ok {
			t := p.Title
			it.ProductTitle = &t
			it.ProductPrice = product.Price{Numeric: p.Price}
		}
		if cover, ok := coverMap[o.ProductID]; ok {
			c := cover
			it.ProductCover = &c
		}
		result = append(result, it)
	}
	return result, nil
}

func timePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func keys(m map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
