package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// stubQuerier + stubTx together implement both the read Querier and the
// transactional txQuerier so the state-machine paths can run without a real DB.
type stubQuerier struct {
	products map[int64]gen.Product
	orders   map[int64]gen.TradeOrder
	users    []gen.ListUsersByIDsRow
	images   []gen.ProductImage

	// counters returned by the "other than" aggregate queries.
	reservedOther int64
	activeOther   int64
	byBuyerActive int64

	pendingOther []gen.TradeOrder
	activeList   []gen.TradeOrder

	// recorded writes
	orderStatusSet map[int64]string
	productStatus  map[int64]string
	completed      []int64
	comments       []gen.InsertProductCommentParams
	insertedOrder  *gen.InsertOrderParams
}

func newStub() *stubQuerier {
	return &stubQuerier{
		products:       map[int64]gen.Product{},
		orders:         map[int64]gen.TradeOrder{},
		orderStatusSet: map[int64]string{},
		productStatus:  map[int64]string{},
	}
}

func (s *stubQuerier) GetProduct(_ context.Context, id int64) (gen.Product, error) {
	p, ok := s.products[id]
	if !ok {
		return gen.Product{}, pgx.ErrNoRows
	}
	return p, nil
}

func (s *stubQuerier) GetOrder(_ context.Context, id int64) (gen.TradeOrder, error) {
	o, ok := s.orders[id]
	if !ok {
		return gen.TradeOrder{}, pgx.ErrNoRows
	}
	return o, nil
}

func (s *stubQuerier) InsertOrder(_ context.Context, arg gen.InsertOrderParams) (gen.TradeOrder, error) {
	s.insertedOrder = &arg
	o := gen.TradeOrder{
		ID: 100, ProductID: arg.ProductID, BuyerID: arg.BuyerID,
		SellerID: arg.SellerID, Status: arg.Status,
		MeetLocation: arg.MeetLocation, Remark: arg.Remark,
	}
	s.orders[o.ID] = o
	return o, nil
}

func (s *stubQuerier) ListOrdersByBuyer(_ context.Context, buyerID int64) ([]gen.TradeOrder, error) {
	var out []gen.TradeOrder
	for _, o := range s.orders {
		if o.BuyerID == buyerID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *stubQuerier) ListOrdersBySeller(_ context.Context, sellerID int64) ([]gen.TradeOrder, error) {
	var out []gen.TradeOrder
	for _, o := range s.orders {
		if o.SellerID == sellerID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *stubQuerier) ListOrdersByUser(_ context.Context, userID int64) ([]gen.TradeOrder, error) {
	var out []gen.TradeOrder
	for _, o := range s.orders {
		if o.BuyerID == userID || o.SellerID == userID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *stubQuerier) CountActiveOrdersByBuyerProduct(_ context.Context, _ gen.CountActiveOrdersByBuyerProductParams) (int64, error) {
	return s.byBuyerActive, nil
}

func (s *stubQuerier) CountReservedOrdersOtherThan(_ context.Context, _ gen.CountReservedOrdersOtherThanParams) (int64, error) {
	return s.reservedOther, nil
}

func (s *stubQuerier) CountActiveOrdersOtherThan(_ context.Context, _ gen.CountActiveOrdersOtherThanParams) (int64, error) {
	return s.activeOther, nil
}

func (s *stubQuerier) ListPendingOrdersOtherThan(_ context.Context, _ gen.ListPendingOrdersOtherThanParams) ([]gen.TradeOrder, error) {
	return s.pendingOther, nil
}

func (s *stubQuerier) ListActiveOrdersOtherThan(_ context.Context, _ gen.ListActiveOrdersOtherThanParams) ([]gen.TradeOrder, error) {
	return s.activeList, nil
}

func (s *stubQuerier) ListProductImages(_ context.Context, _ []int64) ([]gen.ProductImage, error) {
	return s.images, nil
}

func (s *stubQuerier) ListUsersByIDs(_ context.Context, _ []int64) ([]gen.ListUsersByIDsRow, error) {
	return s.users, nil
}

// --- txQuerier methods (recorded writes) ---

func (s *stubQuerier) SetOrderStatus(_ context.Context, arg gen.SetOrderStatusParams) error {
	s.orderStatusSet[arg.ID] = arg.Status
	if o, ok := s.orders[arg.ID]; ok {
		o.Status = arg.Status
		s.orders[arg.ID] = o
	}
	return nil
}

func (s *stubQuerier) SetOrderCompleted(_ context.Context, id int64) error {
	s.completed = append(s.completed, id)
	if o, ok := s.orders[id]; ok {
		o.Status = "COMPLETED"
		o.CompletedAt = pgtype.Timestamptz{Valid: true}
		s.orders[id] = o
	}
	return nil
}

func (s *stubQuerier) SetProductStatus(_ context.Context, arg gen.SetProductStatusParams) error {
	s.productStatus[arg.ID] = arg.Status
	if p, ok := s.products[arg.ID]; ok {
		p.Status = arg.Status
		s.products[arg.ID] = p
	}
	return nil
}

func (s *stubQuerier) InsertProductComment(_ context.Context, arg gen.InsertProductCommentParams) (gen.ProductComment, error) {
	s.comments = append(s.comments, arg)
	return gen.ProductComment{ID: 1, ProductID: arg.ProductID, UserID: arg.UserID, Content: arg.Content}, nil
}

// fakeTx satisfies pgx.Tx for the transactional paths; Rollback/Commit are no-ops.
type fakeTx struct{ pgx.Tx }

func (fakeTx) Commit(context.Context) error   { return nil }
func (fakeTx) Rollback(context.Context) error { return nil }

type fakePool struct{}

func (fakePool) Begin(context.Context) (pgx.Tx, error) { return fakeTx{}, nil }

// newSvc builds a Service whose tx writer routes back to the same stub, so
// writes made inside the transaction are observable on the stub.
func newSvc(stub *stubQuerier) *Service {
	return &Service{
		q:     stub,
		pool:  fakePool{},
		newTx: func(pgx.Tx) txQuerier { return stub },
	}
}

func makeProduct(id, sellerID int64, status string) gen.Product {
	var n pgtype.Numeric
	_ = n.Scan("12.50")
	return gen.Product{ID: id, SellerID: sellerID, CategoryID: 1, Title: "Item", Price: n, OriginalPrice: n, Status: status}
}

func mustBiz(t *testing.T, err error, code int, msg string) {
	t.Helper()
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Code != code || be.Msg != msg {
		t.Fatalf("expected {%d,%q}, got {%d,%q}", code, msg, be.Code, be.Msg)
	}
}

// --- Create preconditions ---

func TestCreateProductNotFound(t *testing.T) {
	stub := newStub()
	pid := int64(1)
	_, err := newSvc(stub).Create(context.Background(), 10, CreateReq{ProductID: &pid})
	mustBiz(t, err, 400, "商品不存在")
}

func TestCreateCannotBuyOwn(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 10, "ON_SALE")
	pid := int64(1)
	_, err := newSvc(stub).Create(context.Background(), 10, CreateReq{ProductID: &pid})
	mustBiz(t, err, 400, "不能购买自己的商品")
}

func TestCreateProductNotOnSale(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "RESERVED")
	pid := int64(1)
	_, err := newSvc(stub).Create(context.Background(), 10, CreateReq{ProductID: &pid})
	mustBiz(t, err, 400, "商品当前不可购买")
}

func TestCreateDuplicateRequest(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "ON_SALE")
	stub.byBuyerActive = 1
	pid := int64(1)
	_, err := newSvc(stub).Create(context.Background(), 10, CreateReq{ProductID: &pid})
	mustBiz(t, err, 400, "你已经提交过预约申请")
}

func TestCreateSuccess(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "ON_SALE")
	stub.users = []gen.ListUsersByIDsRow{{ID: 5, Name: "Seller"}, {ID: 10, Name: "Buyer"}}
	pid := int64(1)
	it, err := newSvc(stub).Create(context.Background(), 10, CreateReq{ProductID: &pid})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if it.Status != "PENDING" || it.BuyerID != 10 || it.SellerID != 5 {
		t.Fatalf("bad item: %+v", it)
	}
	if stub.insertedOrder == nil || stub.insertedOrder.Status != "PENDING" {
		t.Fatalf("insert not recorded correctly")
	}
}

// --- Role guards ---

func TestConfirmWrongUserForbidden(t *testing.T) {
	stub := newStub()
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "PENDING"}
	// buyer (10) tries to confirm; only seller (5) may.
	_, err := newSvc(stub).Confirm(context.Background(), 1, 10)
	mustBiz(t, err, 403, "无权操作")
}

func TestConfirmOrderNotFound(t *testing.T) {
	stub := newStub()
	_, err := newSvc(stub).Confirm(context.Background(), 99, 5)
	mustBiz(t, err, 400, "订单不存在")
}

func TestFinishStrangerForbidden(t *testing.T) {
	stub := newStub()
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "RESERVED"}
	_, err := newSvc(stub).Finish(context.Background(), 1, 999)
	mustBiz(t, err, 403, "无权操作")
}

// --- Transition guards ---

func TestConfirmRequiresPending(t *testing.T) {
	stub := newStub()
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "RESERVED"}
	_, err := newSvc(stub).Confirm(context.Background(), 1, 5)
	mustBiz(t, err, 400, "当前状态不能确认预约")
}

func TestConfirmRejectsWhenAlreadyReserved(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "ON_SALE")
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "PENDING"}
	stub.reservedOther = 1
	_, err := newSvc(stub).Confirm(context.Background(), 1, 5)
	mustBiz(t, err, 400, "该商品已有确认的预约")
}

func TestConfirmSuccessSideEffects(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "ON_SALE")
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "PENDING"}
	stub.orders[2] = gen.TradeOrder{ID: 2, ProductID: 1, BuyerID: 11, SellerID: 5, Status: "PENDING"}
	stub.pendingOther = []gen.TradeOrder{stub.orders[2]}
	stub.users = []gen.ListUsersByIDsRow{{ID: 10, Name: "Buyer"}, {ID: 5, Name: "Seller"}}
	it, err := newSvc(stub).Confirm(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if it.Status != "RESERVED" {
		t.Fatalf("order status = %s", it.Status)
	}
	if stub.productStatus[1] != "RESERVED" {
		t.Fatalf("product status = %s", stub.productStatus[1])
	}
	if stub.orderStatusSet[2] != "CANCELLED" {
		t.Fatalf("sibling pending order not cancelled: %s", stub.orderStatusSet[2])
	}
	if len(stub.comments) != 1 {
		t.Fatalf("expected reservation comment, got %d", len(stub.comments))
	}
}

func TestFinishRequiresReserved(t *testing.T) {
	stub := newStub()
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "PENDING"}
	_, err := newSvc(stub).Finish(context.Background(), 1, 5)
	mustBiz(t, err, 400, "请先由卖家确认预约")
}

func TestFinishSuccessFlipsProductSold(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "RESERVED")
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "RESERVED"}
	stub.users = []gen.ListUsersByIDsRow{{ID: 10, Name: "Buyer"}, {ID: 5, Name: "Seller"}}
	it, err := newSvc(stub).Finish(context.Background(), 1, 10) // buyer may finish
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if it.Status != "COMPLETED" || it.CompletedAt == nil {
		t.Fatalf("bad finish item: %+v", it)
	}
	if stub.productStatus[1] != "SOLD" {
		t.Fatalf("product status = %s", stub.productStatus[1])
	}
	if len(stub.completed) != 1 || stub.completed[0] != 1 {
		t.Fatalf("completed not recorded")
	}
}

func TestCancelCompletedRejected(t *testing.T) {
	stub := newStub()
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "COMPLETED"}
	_, err := newSvc(stub).Cancel(context.Background(), 1, 10)
	mustBiz(t, err, 400, "已完成订单不可取消")
}

func TestCancelRevertsProductWhenNoActive(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "RESERVED")
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "RESERVED"}
	stub.activeOther = 0 // no other active orders
	stub.users = []gen.ListUsersByIDsRow{{ID: 10, Name: "Buyer"}, {ID: 5, Name: "Seller"}}
	it, err := newSvc(stub).Cancel(context.Background(), 1, 5) // seller may cancel
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if it.Status != "CANCELLED" {
		t.Fatalf("order status = %s", it.Status)
	}
	if stub.productStatus[1] != "ON_SALE" {
		t.Fatalf("product should revert to ON_SALE, got %s", stub.productStatus[1])
	}
}

func TestCancelKeepsProductWhenOtherActive(t *testing.T) {
	stub := newStub()
	stub.products[1] = makeProduct(1, 5, "RESERVED")
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "PENDING"}
	stub.activeOther = 1 // another active order remains
	stub.users = []gen.ListUsersByIDsRow{{ID: 10, Name: "Buyer"}, {ID: 5, Name: "Seller"}}
	_, err := newSvc(stub).Cancel(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if _, touched := stub.productStatus[1]; touched {
		t.Fatalf("product status should be untouched, got %s", stub.productStatus[1])
	}
}

// --- MyOrders role filter ---

func TestMyOrdersRoleFilter(t *testing.T) {
	stub := newStub()
	stub.orders[1] = gen.TradeOrder{ID: 1, ProductID: 1, BuyerID: 10, SellerID: 5, Status: "PENDING"}
	stub.orders[2] = gen.TradeOrder{ID: 2, ProductID: 2, BuyerID: 7, SellerID: 10, Status: "PENDING"}
	svc := newSvc(stub)

	buyer, err := svc.MyOrders(context.Background(), 10, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if len(buyer) != 1 || buyer[0].ID != 1 {
		t.Fatalf("buyer role: %+v", buyer)
	}

	seller, err := svc.MyOrders(context.Background(), 10, "seller")
	if err != nil {
		t.Fatal(err)
	}
	if len(seller) != 1 || seller[0].ID != 2 {
		t.Fatalf("seller role: %+v", seller)
	}

	// default/invalid role => both
	all, err := svc.MyOrders(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("default role should return both, got %d", len(all))
	}
}

func TestMyOrdersEmptySerializesSlice(t *testing.T) {
	stub := newStub()
	items, err := newSvc(stub).MyOrders(context.Background(), 999, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("expected non-nil empty slice, got %#v", items)
	}
}
