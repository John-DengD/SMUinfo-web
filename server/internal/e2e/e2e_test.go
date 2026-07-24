// Package e2e is the compatibility acceptance gate for the Go(Gin)+PostgreSQL
// rewrite of SMU Deal. It boots the real router (via app.NewRouter) against a
// freshly-migrated PostgreSQL test database, then drives the full cross-domain
// user journey over HTTP and asserts that every response is byte-compatible
// with the contract the WeChat miniprogram expects: the {code,message,data}
// envelope, camelCase field names, pagination shape {total,records}, and
// 2-decimal price formatting.
//
// The test needs a live database. If TEST_DB_URL is unreachable it skips
// (so `go test ./...` stays green on machines without PostgreSQL). When the DB
// is present it drops+recreates the public schema and reruns migrations, so
// re-running the suite is idempotent.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/John-DengD/smu-deal/server/internal/app"
	"github.com/John-DengD/smu-deal/server/internal/config"
	"github.com/John-DengD/smu-deal/server/internal/db"
)

const defaultTestDSN = "postgres://smu_deal:password@localhost:5432/smu_deal_test?sslmode=disable"

// migrationsDir is resolved relative to this test file (internal/e2e -> server/internal/db/migrations).
const migrationsDir = "../db/migrations"

var (
	srv  *httptest.Server
	pool *pgxpool.Pool
)

func testDSN() string {
	if v := os.Getenv("TEST_DB_URL"); v != "" {
		return v
	}
	return defaultTestDSN
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	dsn := testDSN()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Reachability probe: skip the whole suite if PostgreSQL is not available.
	probe, err := pgxpool.New(ctx, dsn)
	if err == nil {
		err = probe.Ping(ctx)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: skipping, test DB unreachable at %s: %v\n", dsn, err)
		// Exit 0 so the overall `go test ./...` run is not failed by a missing DB.
		// The single test below also guards with t.Skip for `go test ./internal/e2e`.
		os.Exit(0)
	}

	// Clean slate: drop + recreate the public schema, then run all migrations
	// (0001 init, 0002 seed, 0003 transit seed). Idempotent across re-runs.
	if _, err := probe.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: reset schema: %v\n", err)
		os.Exit(1)
	}
	probe.Close()

	if err := db.RunMigrations(dsn, migrationsDir); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: migrate: %v\n", err)
		os.Exit(1)
	}

	pool, err = db.NewPool(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: pool: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Config{
		Port:           "0",
		DBURL:          dsn,
		JWTSecret:      "e2e-test-secret-key-please-do-not-use-in-production-000000",
		JWTExpireHours: 168,
		UploadDir:      os.TempDir(),
		URLPrefix:      "/uploads",
		AllowedOrigins: []string{"http://localhost:5173"},
		MaxFileSize:    5242880,
	}
	engine := app.NewRouter(cfg, pool)
	srv = httptest.NewServer(engine)

	code := m.Run()

	srv.Close()
	pool.Close()
	os.Exit(code)
}

// ---- HTTP helpers -----------------------------------------------------------

// envelope is the compatibility contract. RawData preserves the exact bytes of
// the data field so we can make substring assertions against camelCase names
// and price formatting that survive Go's map ordering.
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// do issues a request and returns the HTTP status, decoded envelope, and the
// full raw response body (for substring/regex assertions).
func do(t *testing.T, method, path, token string, body any) (int, envelope, string) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env envelope
	// Some responses (e.g. the raw 401 body) are still valid envelopes.
	_ = json.Unmarshal(raw, &env)
	return resp.StatusCode, env, string(raw)
}

// okData issues a request, asserts HTTP 200 + envelope code==0, and returns the
// raw data bytes plus the whole body string for further assertions.
func okData(t *testing.T, method, path, token string, body any) (json.RawMessage, string) {
	t.Helper()
	status, env, raw := do(t, method, path, token, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: expected HTTP 200, got %d (body=%s)", method, path, status, raw)
	}
	if env.Code != 0 {
		t.Fatalf("%s %s: expected envelope code 0, got %d message=%q (body=%s)", method, path, env.Code, env.Message, raw)
	}
	if env.Message != "ok" {
		t.Fatalf("%s %s: expected message \"ok\", got %q", method, path, env.Message)
	}
	// Every success envelope must carry all three top-level keys.
	if !strings.Contains(raw, `"code"`) || !strings.Contains(raw, `"message"`) || !strings.Contains(raw, `"data"`) {
		t.Fatalf("%s %s: envelope missing top-level keys (body=%s)", method, path, raw)
	}
	return env.Data, raw
}

func decode(t *testing.T, data json.RawMessage, into any) {
	t.Helper()
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("decode data: %v (data=%s)", err, string(data))
	}
}

func mustContain(t *testing.T, raw, sub, what string) {
	t.Helper()
	if !strings.Contains(raw, sub) {
		t.Fatalf("%s: expected raw JSON to contain %q\nbody=%s", what, sub, raw)
	}
}

// ---- shared state across ordered subtests -----------------------------------

var st struct {
	tokenA, tokenB, tokenAdmin string
	userAID, userBID           int64
	productID                  int64
	orderID                    int64
	reportID                   int64
	feedbackID                 int64
	annID                      int64
}

// TestCompatibilityJourney is one ordered test: each subtest depends on prior
// state. If the DB is unavailable it skips (belt-and-suspenders with TestMain).
func TestCompatibilityJourney(t *testing.T) {
	if srv == nil {
		t.Skip("e2e: no server (test DB unavailable)")
	}

	t.Run("register user A", func(t *testing.T) {
		data, raw := okData(t, "POST", "/api/auth/register", "", map[string]any{
			"name":      "用户甲",
			"studentNo": "201900010001",
			"password":  "password123",
			"college":   "计算机学院",
			"campus":    "武功山校区",
		})
		mustContain(t, raw, `"studentNo"`, "register camelCase studentNo")
		if strings.Contains(raw, `"student_no"`) {
			t.Fatalf("register: snake_case student_no leaked: %s", raw)
		}
		var u struct {
			ID        int64  `json:"id"`
			StudentNo string `json:"studentNo"`
			Role      string `json:"role"`
			Status    string `json:"status"`
		}
		decode(t, data, &u)
		if u.ID == 0 || u.StudentNo != "201900010001" {
			t.Fatalf("register A: bad user %+v", u)
		}
		if u.Role != "USER" || u.Status != "ACTIVE" {
			t.Fatalf("register A: expected USER/ACTIVE got %s/%s", u.Role, u.Status)
		}
		st.userAID = u.ID
	})

	t.Run("login user A", func(t *testing.T) {
		data, raw := okData(t, "POST", "/api/auth/login", "", map[string]any{
			"studentNo": "201900010001",
			"password":  "password123",
		})
		mustContain(t, raw, `"token"`, "login token")
		mustContain(t, raw, `"user"`, "login user object")
		mustContain(t, raw, `"studentNo"`, "login user camelCase")
		var lr struct {
			Token string `json:"token"`
			User  struct {
				ID        int64  `json:"id"`
				StudentNo string `json:"studentNo"`
			} `json:"user"`
		}
		decode(t, data, &lr)
		if lr.Token == "" {
			t.Fatal("login A: empty token")
		}
		if lr.User.ID != st.userAID {
			t.Fatalf("login A: user id mismatch %d != %d", lr.User.ID, st.userAID)
		}
		st.tokenA = lr.Token
	})

	t.Run("register + login user B", func(t *testing.T) {
		data, _ := okData(t, "POST", "/api/auth/register", "", map[string]any{
			"name":      "用户乙",
			"studentNo": "201900010002",
			"password":  "password123",
			"campus":    "武功山校区",
		})
		var u struct {
			ID int64 `json:"id"`
		}
		decode(t, data, &u)
		st.userBID = u.ID

		ld, _ := okData(t, "POST", "/api/auth/login", "", map[string]any{
			"studentNo": "201900010002",
			"password":  "password123",
		})
		var lr struct {
			Token string `json:"token"`
		}
		decode(t, ld, &lr)
		st.tokenB = lr.Token
		if st.tokenB == "" {
			t.Fatal("login B: empty token")
		}
	})

	t.Run("unauthenticated protected route -> 401", func(t *testing.T) {
		status, env, raw := do(t, "GET", "/api/users/me", "", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("GET /api/users/me no token: expected HTTP 401, got %d (body=%s)", status, raw)
		}
		if env.Code != 401 {
			t.Fatalf("GET /api/users/me no token: expected envelope code 401, got %d (body=%s)", env.Code, raw)
		}
	})

	t.Run("categories public list (9 seeded, camelCase sortOrder)", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/categories", "", nil)
		mustContain(t, raw, `"sortOrder"`, "categories camelCase sortOrder")
		if strings.Contains(raw, `"sort_order"`) {
			t.Fatalf("categories: snake_case sort_order leaked: %s", raw)
		}
		var cats []struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			SortOrder int32  `json:"sortOrder"`
		}
		decode(t, data, &cats)
		if len(cats) != 9 {
			t.Fatalf("categories: expected 9 seeded categories, got %d", len(cats))
		}
	})

	t.Run("A creates product", func(t *testing.T) {
		data, raw := okData(t, "POST", "/api/products", st.tokenA, map[string]any{
			"title":          "二手教材 数据结构",
			"description":    "九成新",
			"categoryId":     1,
			"price":          12.50,
			"originalPrice":  40.00,
			"conditionLevel": "GOOD",
			"tradeLocation":  "图书馆门口",
			"images":         []string{"/uploads/a.jpg", "/uploads/b.jpg"},
		})
		mustContain(t, raw, `"viewCount"`, "product create camelCase viewCount")
		mustContain(t, raw, `"categoryId"`, "product create camelCase categoryId")
		var p struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
			Images []string `json:"images"`
		}
		decode(t, data, &p)
		if p.ID == 0 {
			t.Fatalf("create product: no id (body=%s)", raw)
		}
		if p.Status != "ON_SALE" {
			t.Fatalf("create product: expected ON_SALE, got %s", p.Status)
		}
		if len(p.Images) != 2 {
			t.Fatalf("create product: expected 2 images, got %d", len(p.Images))
		}
		st.productID = p.ID
	})

	t.Run("products list pagination shape + price 2dp", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/products", "", nil)
		mustContain(t, raw, `"total"`, "products pagination total")
		mustContain(t, raw, `"records"`, "products pagination records")
		mustContain(t, raw, `"viewCount"`, "products camelCase viewCount")
		mustContain(t, raw, `"categoryName"`, "products camelCase categoryName")
		mustContain(t, raw, `"sellerName"`, "products camelCase sellerName")
		// Price must be a JSON number with exactly 2 decimals.
		mustContain(t, raw, `"price":12.50`, "products price formatted 2 decimals")
		var page struct {
			Total   int64 `json:"total"`
			Records []struct {
				ID           int64  `json:"id"`
				ViewCount    int32  `json:"viewCount"`
				CategoryName string `json:"categoryName"`
				SellerName   string `json:"sellerName"`
			} `json:"records"`
		}
		decode(t, data, &page)
		if page.Total < 1 || len(page.Records) < 1 {
			t.Fatalf("products list: empty page total=%d records=%d", page.Total, len(page.Records))
		}
	})

	t.Run("product detail increments view_count", func(t *testing.T) {
		d1, _ := okData(t, "GET", fmt.Sprintf("/api/products/%d", st.productID), "", nil)
		var v1 struct {
			ViewCount int32 `json:"viewCount"`
		}
		decode(t, d1, &v1)
		d2, _ := okData(t, "GET", fmt.Sprintf("/api/products/%d", st.productID), "", nil)
		var v2 struct {
			ViewCount int32 `json:"viewCount"`
		}
		decode(t, d2, &v2)
		if v2.ViewCount <= v1.ViewCount {
			t.Fatalf("view_count did not increment across calls: %d -> %d", v1.ViewCount, v2.ViewCount)
		}
	})

	t.Run("B favorites then unfavorites", func(t *testing.T) {
		okData(t, "POST", fmt.Sprintf("/api/favorites/%d", st.productID), st.tokenB, nil)
		data, raw := okData(t, "GET", "/api/favorites", st.tokenB, nil)
		mustContain(t, raw, `"favorited":true`, "favorites favorited:true")
		mustContain(t, raw, `"total"`, "favorites pagination shape")
		var page struct {
			Total   int64 `json:"total"`
			Records []struct {
				ID        int64 `json:"id"`
				Favorited bool  `json:"favorited"`
			} `json:"records"`
		}
		decode(t, data, &page)
		if page.Total < 1 {
			t.Fatalf("favorites: expected >=1 favorite, got total=%d", page.Total)
		}
		found := false
		for _, r := range page.Records {
			if r.ID == st.productID && r.Favorited {
				found = true
			}
		}
		if !found {
			t.Fatalf("favorites: product %d not present/favorited (body=%s)", st.productID, raw)
		}

		okData(t, "DELETE", fmt.Sprintf("/api/favorites/%d", st.productID), st.tokenB, nil)
		d2, _ := okData(t, "GET", "/api/favorites", st.tokenB, nil)
		var page2 struct {
			Total int64 `json:"total"`
		}
		decode(t, d2, &page2)
		if page2.Total != 0 {
			t.Fatalf("favorites after delete: expected 0, got %d", page2.Total)
		}
	})

	t.Run("order lifecycle PENDING->RESERVED->COMPLETED", func(t *testing.T) {
		// B creates order for A's product.
		data, raw := okData(t, "POST", "/api/orders", st.tokenB, map[string]any{
			"productId":    st.productID,
			"meetLocation": "图书馆",
			"remark":       "下午三点",
		})
		mustContain(t, raw, `"buyerId"`, "order camelCase buyerId")
		mustContain(t, raw, `"sellerId"`, "order camelCase sellerId")
		mustContain(t, raw, `"createdAt"`, "order camelCase createdAt")
		var o struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		}
		decode(t, data, &o)
		if o.Status != "PENDING" {
			t.Fatalf("order create: expected PENDING, got %s", o.Status)
		}
		st.orderID = o.ID

		// A confirms -> RESERVED.
		cd, _ := okData(t, "PUT", fmt.Sprintf("/api/orders/%d/confirm", st.orderID), st.tokenA, nil)
		var oc struct {
			Status string `json:"status"`
		}
		decode(t, cd, &oc)
		if oc.Status != "RESERVED" {
			t.Fatalf("order confirm: expected RESERVED, got %s", oc.Status)
		}
		// Product should now be RESERVED.
		pd, _ := okData(t, "GET", fmt.Sprintf("/api/products/%d", st.productID), "", nil)
		var ps struct {
			Status string `json:"status"`
		}
		decode(t, pd, &ps)
		if ps.Status != "RESERVED" {
			t.Fatalf("product after confirm: expected RESERVED, got %s", ps.Status)
		}

		// A finishes -> COMPLETED + completedAt set.
		fd, fraw := okData(t, "PUT", fmt.Sprintf("/api/orders/%d/finish", st.orderID), st.tokenA, nil)
		mustContain(t, fraw, `"completedAt"`, "order camelCase completedAt")
		var of struct {
			Status      string  `json:"status"`
			CompletedAt *string `json:"completedAt"`
		}
		decode(t, fd, &of)
		if of.Status != "COMPLETED" {
			t.Fatalf("order finish: expected COMPLETED, got %s", of.Status)
		}
		if of.CompletedAt == nil || *of.CompletedAt == "" {
			t.Fatalf("order finish: completedAt not set (body=%s)", fraw)
		}
		// Product should now be SOLD.
		pd2, _ := okData(t, "GET", fmt.Sprintf("/api/products/%d", st.productID), "", nil)
		var ps2 struct {
			Status string `json:"status"`
		}
		decode(t, pd2, &ps2)
		if ps2.Status != "SOLD" {
			t.Fatalf("product after finish: expected SOLD, got %s", ps2.Status)
		}
	})

	t.Run("messaging flow", func(t *testing.T) {
		// B sends a message to A.
		_, sraw := okData(t, "POST", "/api/messages", st.tokenB, map[string]any{
			"receiverId": st.userAID,
			"productId":  st.productID,
			"content":    "你好，教材还在吗？",
		})
		mustContain(t, sraw, `"receiverId"`, "message camelCase receiverId")
		mustContain(t, sraw, `"createdAt"`, "message camelCase createdAt")

		// A unread-count == 1.
		ud, uraw := okData(t, "GET", "/api/messages/unread-count", st.tokenA, nil)
		mustContain(t, uraw, `"count"`, "unread-count count field")
		var uc struct {
			Count int `json:"count"`
		}
		decode(t, ud, &uc)
		if uc.Count != 1 {
			t.Fatalf("unread-count: expected 1, got %d (body=%s)", uc.Count, uraw)
		}

		// A conversation list with peerId + unreadCount.
		cd, craw := okData(t, "GET", "/api/messages", st.tokenA, nil)
		mustContain(t, craw, `"peerId"`, "conversation camelCase peerId")
		mustContain(t, craw, `"unreadCount"`, "conversation camelCase unreadCount")
		var convs []struct {
			PeerID      int64 `json:"peerId"`
			UnreadCount int   `json:"unreadCount"`
		}
		decode(t, cd, &convs)
		if len(convs) < 1 || convs[0].PeerID != st.userBID {
			t.Fatalf("conversation: expected peer B=%d, got %+v", st.userBID, convs)
		}

		// A reads the thread -> unread goes to 0.
		okData(t, "GET", fmt.Sprintf("/api/messages/conversation/%d", st.userBID), st.tokenA, nil)
		ud2, _ := okData(t, "GET", "/api/messages/unread-count", st.tokenA, nil)
		var uc2 struct {
			Count int `json:"count"`
		}
		decode(t, ud2, &uc2)
		if uc2.Count != 0 {
			t.Fatalf("unread-count after read: expected 0, got %d", uc2.Count)
		}
	})

	t.Run("lost-found create + public list + detail", func(t *testing.T) {
		data, raw := okData(t, "POST", "/api/lost-found", st.tokenA, map[string]any{
			"type":        "LOST",
			"title":       "丢失校园卡",
			"description": "在图书馆丢失一张校园卡",
			"location":    "图书馆",
			"contact":     "微信 abc",
			"images":      []string{"/uploads/lf.jpg"},
		})
		mustContain(t, raw, `"viewCount"`, "lost-found camelCase viewCount")
		mustContain(t, raw, `"typeText"`, "lost-found camelCase typeText")
		var lf struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		}
		decode(t, data, &lf)
		if lf.ID == 0 || lf.Type != "LOST" {
			t.Fatalf("lost-found create: bad item %+v", lf)
		}

		ld, lraw := okData(t, "GET", "/api/lost-found", "", nil)
		mustContain(t, lraw, `"total"`, "lost-found list pagination total")
		mustContain(t, lraw, `"records"`, "lost-found list pagination records")
		var page struct {
			Total   int64 `json:"total"`
			Records []any `json:"records"`
		}
		decode(t, ld, &page)
		if page.Total < 1 {
			t.Fatalf("lost-found list: expected >=1, got %d", page.Total)
		}

		dd, draw := okData(t, "GET", fmt.Sprintf("/api/lost-found/%d", lf.ID), "", nil)
		mustContain(t, draw, `"images"`, "lost-found detail images")
		var det struct {
			ID     int64    `json:"id"`
			Images []string `json:"images"`
		}
		decode(t, dd, &det)
		if det.ID != lf.ID || len(det.Images) != 1 {
			t.Fatalf("lost-found detail: bad %+v", det)
		}
	})

	t.Run("announcements active -> null (not a list)", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/announcements/active", "", nil)
		// data must be null (no active announcement seeded) — NOT an empty array.
		if strings.TrimSpace(string(data)) != "null" {
			t.Fatalf("announcements/active: expected data null, got %s", string(data))
		}
		if strings.Contains(raw, "[") {
			t.Fatalf("announcements/active: must not be a list (body=%s)", raw)
		}
	})

	t.Run("B reports the product", func(t *testing.T) {
		okData(t, "POST", "/api/reports", st.tokenB, map[string]any{
			"productId": st.productID,
			"reason":    "疑似虚假信息",
		})
	})

	t.Run("B submits feedback + lists mine", func(t *testing.T) {
		okData(t, "POST", "/api/feedback", st.tokenB, map[string]any{
			"category": "功能建议",
			"content":  "希望增加夜间模式",
			"contact":  "qq 123456",
		})
		data, raw := okData(t, "GET", "/api/feedback/mine", st.tokenB, nil)
		mustContain(t, raw, `"createdAt"`, "feedback camelCase createdAt")
		mustContain(t, raw, `"adminReply"`, "feedback camelCase adminReply")
		var items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		}
		decode(t, data, &items)
		if len(items) < 1 {
			t.Fatalf("feedback/mine: expected >=1, got %d", len(items))
		}
		st.feedbackID = items[0].ID
	})

	t.Run("transit next public camelCase", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/transit/next", "", nil)
		mustContain(t, raw, `"lineName"`, "transit camelCase lineName")
		mustContain(t, raw, `"nearest"`, "transit nearest field")
		mustContain(t, raw, `"now"`, "transit now field")
		var resp struct {
			Now      string `json:"now"`
			Line     string `json:"line"`
			LineName string `json:"lineName"`
			Nearest  *struct {
				Time string `json:"time"`
			} `json:"nearest"`
		}
		decode(t, data, &resp)
		if resp.Now == "" || resp.LineName == "" {
			t.Fatalf("transit next: missing now/lineName (body=%s)", raw)
		}
		if resp.Nearest != nil && resp.Nearest.Time == "" {
			t.Fatalf("transit next: nearest present but empty time (body=%s)", raw)
		}
	})

	t.Run("admin login (seed admin)", func(t *testing.T) {
		data, _ := okData(t, "POST", "/api/auth/login", "", map[string]any{
			"studentNo": "admin",
			"password":  "admin123",
		})
		var lr struct {
			Token string `json:"token"`
			User  struct {
				Role string `json:"role"`
			} `json:"user"`
		}
		decode(t, data, &lr)
		if lr.Token == "" || lr.User.Role != "ADMIN" {
			t.Fatalf("admin login: bad %+v", lr)
		}
		st.tokenAdmin = lr.Token
	})

	t.Run("non-admin on admin route -> 403 无权访问", func(t *testing.T) {
		status, env, raw := do(t, "GET", "/api/admin/users", st.tokenB, nil)
		if status != http.StatusForbidden {
			t.Fatalf("admin route with user token: expected HTTP 403, got %d (body=%s)", status, raw)
		}
		if env.Code != 403 {
			t.Fatalf("admin route with user token: expected code 403, got %d", env.Code)
		}
		if !strings.Contains(env.Message, "无权访问") {
			t.Fatalf("admin route with user token: expected message 无权访问, got %q", env.Message)
		}
	})

	t.Run("admin users list + status toggle", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/admin/users", st.tokenAdmin, nil)
		mustContain(t, raw, `"total"`, "admin users pagination total")
		mustContain(t, raw, `"records"`, "admin users pagination records")
		mustContain(t, raw, `"studentNo"`, "admin users camelCase studentNo")
		var page struct {
			Total int64 `json:"total"`
		}
		decode(t, data, &page)
		if page.Total < 3 { // A, B, admin
			t.Fatalf("admin users: expected >=3, got %d", page.Total)
		}

		// Disable then enable B.
		okData(t, "PUT", fmt.Sprintf("/api/admin/users/%d/status", st.userBID), st.tokenAdmin,
			map[string]any{"status": "DISABLED"})
		// A disabled B cannot log in.
		st2, envB, _ := do(t, "POST", "/api/auth/login", "", map[string]any{
			"studentNo": "201900010002", "password": "password123",
		})
		if st2 != http.StatusOK || envB.Code == 0 {
			t.Fatalf("disabled B login: expected business failure, got status=%d code=%d", st2, envB.Code)
		}
		okData(t, "PUT", fmt.Sprintf("/api/admin/users/%d/status", st.userBID), st.tokenAdmin,
			map[string]any{"status": "ACTIVE"})
	})

	t.Run("admin products (OFFLINE visible) + status change", func(t *testing.T) {
		// Take product OFFLINE via admin, then confirm it is still visible on the
		// admin product list (which includes all statuses).
		okData(t, "PUT", fmt.Sprintf("/api/admin/products/%d/status", st.productID), st.tokenAdmin,
			map[string]any{"status": "OFFLINE"})
		data, raw := okData(t, "GET", "/api/admin/products", st.tokenAdmin, nil)
		mustContain(t, raw, `"total"`, "admin products pagination")
		var page struct {
			Records []struct {
				ID     int64  `json:"id"`
				Status string `json:"status"`
			} `json:"records"`
		}
		decode(t, data, &page)
		found := false
		for _, r := range page.Records {
			if r.ID == st.productID {
				found = true
				if r.Status != "OFFLINE" {
					t.Fatalf("admin products: product status expected OFFLINE, got %s", r.Status)
				}
			}
		}
		if !found {
			t.Fatalf("admin products: OFFLINE product %d not visible (body=%s)", st.productID, raw)
		}
	})

	t.Run("admin reports list + resolve", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/admin/reports", st.tokenAdmin, nil)
		mustContain(t, raw, `"reporterId"`, "admin reports camelCase reporterId")
		mustContain(t, raw, `"adminRemark"`, "admin reports camelCase adminRemark")
		var items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		}
		decode(t, data, &items)
		if len(items) < 1 {
			t.Fatalf("admin reports: expected >=1, got %d", len(items))
		}
		st.reportID = items[0].ID
		okData(t, "PUT", fmt.Sprintf("/api/admin/reports/%d", st.reportID), st.tokenAdmin,
			map[string]any{"status": "RESOLVED", "adminRemark": "已处理"})
	})

	t.Run("admin feedback list + reply", func(t *testing.T) {
		data, raw := okData(t, "GET", "/api/admin/feedback", st.tokenAdmin, nil)
		mustContain(t, raw, `"adminReply"`, "admin feedback camelCase adminReply")
		var items []struct {
			ID int64 `json:"id"`
		}
		decode(t, data, &items)
		if len(items) < 1 {
			t.Fatalf("admin feedback: expected >=1, got %d", len(items))
		}
		okData(t, "PUT", fmt.Sprintf("/api/admin/feedback/%d", items[0].ID), st.tokenAdmin,
			map[string]any{"status": "RESOLVED", "adminReply": "感谢反馈"})
	})

	t.Run("admin announcement CRUD + active reflects it", func(t *testing.T) {
		// Create ACTIVE announcement.
		data, raw := okData(t, "POST", "/api/admin/announcements", st.tokenAdmin, map[string]any{
			"title":   "系统维护通知",
			"content":  "本周日凌晨维护",
			"status":  "ACTIVE",
		})
		mustContain(t, raw, `"createdAt"`, "announcement camelCase createdAt")
		var ann struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		}
		decode(t, data, &ann)
		if ann.ID == 0 {
			t.Fatalf("create announcement: no id (body=%s)", raw)
		}
		st.annID = ann.ID

		// Public active now returns the single item (not a list).
		ad, araw := okData(t, "GET", "/api/announcements/active", "", nil)
		if strings.TrimSpace(string(ad)) == "null" {
			t.Fatalf("announcements/active: expected item after create, got null")
		}
		if strings.HasPrefix(strings.TrimSpace(string(ad)), "[") {
			t.Fatalf("announcements/active: must be single object not list (body=%s)", araw)
		}
		var active struct {
			ID int64 `json:"id"`
		}
		decode(t, ad, &active)
		if active.ID != st.annID {
			t.Fatalf("announcements/active: id mismatch %d != %d", active.ID, st.annID)
		}

		// Admin list.
		ld, _ := okData(t, "GET", "/api/admin/announcements", st.tokenAdmin, nil)
		var list []struct {
			ID int64 `json:"id"`
		}
		decode(t, ld, &list)
		if len(list) < 1 {
			t.Fatalf("admin announcements list: expected >=1, got %d", len(list))
		}

		// Update.
		okData(t, "PUT", fmt.Sprintf("/api/admin/announcements/%d", st.annID), st.tokenAdmin,
			map[string]any{"title": "系统维护通知(更新)", "content": "延期至下周", "status": "ACTIVE"})

		// Delete.
		okData(t, "DELETE", fmt.Sprintf("/api/admin/announcements/%d", st.annID), st.tokenAdmin, nil)
		ad2, _ := okData(t, "GET", "/api/announcements/active", "", nil)
		if strings.TrimSpace(string(ad2)) != "null" {
			t.Fatalf("announcements/active after delete: expected null, got %s", string(ad2))
		}
	})
}
