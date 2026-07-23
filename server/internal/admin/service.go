package admin

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
// Only the announcement create/update paths need it, since each performs two
// writes (insert/update + disableOtherActive) that must be atomic to preserve
// the single-active invariant, matching Java @Transactional. Tests that don't
// exercise those paths may pass a nil pool.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// annTxQuerier is the subset of writes the transactional announcement paths run
// inside a transaction. Both *gen.Queries (production) and the test stub satisfy it.
type annTxQuerier interface {
	InsertAnnouncement(ctx context.Context, arg gen.InsertAnnouncementParams) (gen.Announcement, error)
	UpdateAnnouncement(ctx context.Context, arg gen.UpdateAnnouncementParams) (gen.Announcement, error)
	DisableOtherActiveAnnouncements(ctx context.Context, id int64) error
}

// Querier is the subset of the sqlc-generated *gen.Queries the admin service needs.
type Querier interface {
	// users
	CountAdminUsers(ctx context.Context, keyword *string) (int64, error)
	ListAdminUsers(ctx context.Context, arg gen.ListAdminUsersParams) ([]gen.ListAdminUsersRow, error)
	GetUserByID(ctx context.Context, id int64) (gen.User, error)
	SetUserStatus(ctx context.Context, arg gen.SetUserStatusParams) error

	// categories
	ListActiveCategories(ctx context.Context) ([]gen.Category, error)
	GetCategory(ctx context.Context, id int64) (gen.Category, error)
	InsertCategory(ctx context.Context, arg gen.InsertCategoryParams) (gen.Category, error)
	UpdateCategory(ctx context.Context, arg gen.UpdateCategoryParams) (gen.Category, error)
	DeleteCategory(ctx context.Context, id int64) error

	// reports
	ListReports(ctx context.Context, status *string) ([]gen.Report, error)
	GetReport(ctx context.Context, id int64) (gen.Report, error)
	UpdateReport(ctx context.Context, arg gen.UpdateReportParams) error
	ListUsersByIDs(ctx context.Context, ids []int64) ([]gen.ListUsersByIDsRow, error)
	ListProductTitlesByIDs(ctx context.Context, ids []int64) ([]gen.ListProductTitlesByIDsRow, error)

	// feedback
	ListAllFeedback(ctx context.Context, status *string) ([]gen.Feedback, error)
	GetFeedback(ctx context.Context, id int64) (gen.Feedback, error)
	UpdateFeedbackReply(ctx context.Context, arg gen.UpdateFeedbackReplyParams) error

	// announcements
	ListAllAnnouncements(ctx context.Context) ([]gen.Announcement, error)
	GetAnnouncement(ctx context.Context, id int64) (gen.Announcement, error)
	InsertAnnouncement(ctx context.Context, arg gen.InsertAnnouncementParams) (gen.Announcement, error)
	UpdateAnnouncement(ctx context.Context, arg gen.UpdateAnnouncementParams) (gen.Announcement, error)
	DeleteAnnouncement(ctx context.Context, id int64) error
	DisableOtherActiveAnnouncements(ctx context.Context, id int64) error
}

type Service struct {
	q       Querier
	product *product.Service
	pool    TxBeginner
	// newAnnTx builds the tx-scoped announcement writer from a begun transaction.
	// In production it returns gen.New(tx); tests can override it to route tx
	// writes back to their stub without a real DB.
	newAnnTx func(tx pgx.Tx) annTxQuerier
}

// NewService constructs the admin service. The product service is reused for
// admin product listing (includeAllStatus) and force-status changes, matching
// the Java AdminController delegating to ProductService. pool is used only by
// the announcement create/update paths to run their two writes atomically; in
// production it is the same *pgxpool.Pool backing q.
func NewService(q Querier, prod *product.Service, pool TxBeginner) *Service {
	return &Service{
		q:        q,
		product:  prod,
		pool:     pool,
		newAnnTx: func(tx pgx.Tx) annTxQuerier { return gen.New(tx) },
	}
}

// --- users ---

// UserInfo mirrors AuthDTO.UserInfo (camelCase wire contract). Field order and
// nullability match the Java AdminUserService.list mapping.
type UserInfo struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	StudentNo string  `json:"studentNo"`
	Phone     *string `json:"phone"`
	College   *string `json:"college"`
	Campus    *string `json:"campus"`
	Avatar    *string `json:"avatar"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
}

// ListUsers replicates AdminUserService.list: keyword LIKE over name/studentNo/phone,
// order by created_at desc, page (default 1) / size (default 20) pagination.
func (s *Service) ListUsers(ctx context.Context, keyword *string, page, size *int32) (httpx.Page, error) {
	p := int32(1)
	if page != nil {
		p = *page
	}
	sz := int32(20)
	if size != nil {
		sz = *size
	}
	if p < 1 {
		p = 1
	}
	if sz < 1 {
		sz = 20
	}
	if sz > 100 {
		sz = 100
	}
	off64 := (int64(p) - 1) * int64(sz)
	if off64 > int64(^uint32(0)>>1) {
		off64 = int64(^uint32(0) >> 1)
	}
	offset := int32(off64)

	kw := blankToNil(keyword)

	total, err := s.q.CountAdminUsers(ctx, kw)
	if err != nil {
		return httpx.Page{}, err
	}
	rows, err := s.q.ListAdminUsers(ctx, gen.ListAdminUsersParams{Keyword: kw, Off: offset, Lim: sz})
	if err != nil {
		return httpx.Page{}, err
	}
	records := make([]UserInfo, 0, len(rows))
	for _, u := range rows {
		records = append(records, UserInfo{
			ID:        u.ID,
			Name:      u.Name,
			StudentNo: u.StudentNo,
			Phone:     u.Phone,
			College:   u.College,
			Campus:    u.Campus,
			Avatar:    u.Avatar,
			Role:      u.Role,
			Status:    u.Status,
		})
	}
	return httpx.Page{Total: total, Records: records}, nil
}

// ChangeUserStatus replicates AdminUserService.changeStatus: 404-style business
// error if the user is missing, otherwise sets the new status verbatim.
func (s *Service) ChangeUserStatus(ctx context.Context, id int64, status string) error {
	if _, err := s.q.GetUserByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("用户不存在")
		}
		return err
	}
	return s.q.SetUserStatus(ctx, gen.SetUserStatusParams{ID: id, Status: status})
}

// --- products (delegated to product.Service) ---

// ListProducts replicates AdminController.products: forces includeAllStatus=true
// then delegates to ProductService.list with the admin as current user.
func (s *Service) ListProducts(ctx context.Context, q product.ListQuery, adminID int64) (httpx.Page, error) {
	includeAll := true
	q.IncludeAllStatus = &includeAll
	return s.product.List(ctx, q, &adminID)
}

// ChangeProductStatus replicates AdminController.productStatus: delegates to
// ProductService.changeStatus with isAdmin=true (bypasses owner check).
func (s *Service) ChangeProductStatus(ctx context.Context, id, adminID int64, status string) error {
	return s.product.ChangeStatus(ctx, id, adminID, true, status)
}

// --- categories ---

// Category mirrors the Category entity wire contract (camelCase JSON).
type Category struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Icon      *string    `json:"icon"`
	SortOrder int32      `json:"sortOrder"`
	Status    string     `json:"status"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

func categoryToItem(c gen.Category) Category {
	return Category{
		ID:        c.ID,
		Name:      c.Name,
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
		Status:    c.Status,
		CreatedAt: timePtr(c.CreatedAt),
		UpdatedAt: timePtr(c.UpdatedAt),
	}
}

// CategoryReq mirrors the Category entity used as the create/update request body.
type CategoryReq struct {
	Name      string  `json:"name"`
	Icon      *string `json:"icon"`
	SortOrder *int32  `json:"sortOrder"`
	Status    *string `json:"status"`
}

// ListCategories replicates AdminController.categories -> CategoryService.list:
// only ACTIVE categories, ordered by sort_order ascending.
func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.q.ListActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Category, 0, len(rows))
	for _, r := range rows {
		items = append(items, categoryToItem(r))
	}
	return items, nil
}

// CreateCategory replicates CategoryService.create: status defaults to ACTIVE,
// sortOrder defaults to 0.
func (s *Service) CreateCategory(ctx context.Context, req CategoryReq) (Category, error) {
	status := "ACTIVE"
	if req.Status != nil {
		status = *req.Status
	}
	sortOrder := int32(0)
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	c, err := s.q.InsertCategory(ctx, gen.InsertCategoryParams{
		Name:      req.Name,
		Icon:      req.Icon,
		SortOrder: sortOrder,
		Status:    status,
	})
	if err != nil {
		return Category{}, err
	}
	return categoryToItem(c), nil
}

// UpdateCategory replicates CategoryService.update: MyBatis-Plus updateById only
// writes non-null columns, then re-selects the row. We load the existing row,
// overlay provided fields, and persist so untouched columns are preserved.
func (s *Service) UpdateCategory(ctx context.Context, id int64, req CategoryReq) (Category, error) {
	existing, err := s.q.GetCategory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Category{}, httpx.Biz("分类不存在")
		}
		return Category{}, err
	}
	name := existing.Name
	if strings.TrimSpace(req.Name) != "" {
		name = req.Name
	}
	icon := existing.Icon
	if req.Icon != nil {
		icon = req.Icon
	}
	sortOrder := existing.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}
	c, err := s.q.UpdateCategory(ctx, gen.UpdateCategoryParams{
		ID:        id,
		Name:      name,
		Icon:      icon,
		SortOrder: sortOrder,
		Status:    status,
	})
	if err != nil {
		return Category{}, err
	}
	return categoryToItem(c), nil
}

// DeleteCategory replicates CategoryService.delete: hard delete by id, no guard.
func (s *Service) DeleteCategory(ctx context.Context, id int64) error {
	return s.q.DeleteCategory(ctx, id)
}

// --- reports ---

// ReportItem mirrors ReportDTO.Item (camelCase wire contract).
type ReportItem struct {
	ID           int64      `json:"id"`
	ReporterID   int64      `json:"reporterId"`
	ReporterName *string    `json:"reporterName"`
	ProductID    int64      `json:"productId"`
	ProductTitle *string    `json:"productTitle"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	AdminRemark  *string    `json:"adminRemark"`
	CreatedAt    *time.Time `json:"createdAt"`
}

// ReportHandleReq mirrors ReportDTO.HandleReq.
type ReportHandleReq struct {
	Status      *string `json:"status"`
	AdminRemark *string `json:"adminRemark"`
}

// ListReports replicates ReportService.list: optional status filter, ordered by
// created_at desc, enriched with reporterName and productTitle.
func (s *Service) ListReports(ctx context.Context, status *string) ([]ReportItem, error) {
	rows, err := s.q.ListReports(ctx, blankToNil(status))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return make([]ReportItem, 0), nil
	}
	userSet := map[int64]struct{}{}
	prodSet := map[int64]struct{}{}
	for _, r := range rows {
		userSet[r.ReporterID] = struct{}{}
		prodSet[r.ProductID] = struct{}{}
	}
	users, err := s.q.ListUsersByIDs(ctx, keys(userSet))
	if err != nil {
		return nil, err
	}
	userMap := map[int64]string{}
	for _, u := range users {
		userMap[u.ID] = u.Name
	}
	prods, err := s.q.ListProductTitlesByIDs(ctx, keys(prodSet))
	if err != nil {
		return nil, err
	}
	prodMap := map[int64]string{}
	for _, p := range prods {
		prodMap[p.ID] = p.Title
	}
	items := make([]ReportItem, 0, len(rows))
	for _, r := range rows {
		it := ReportItem{
			ID:          r.ID,
			ReporterID:  r.ReporterID,
			ProductID:   r.ProductID,
			Reason:      r.Reason,
			Status:      r.Status,
			AdminRemark: r.AdminRemark,
			CreatedAt:   timePtr(r.CreatedAt),
		}
		if name, ok := userMap[r.ReporterID]; ok {
			n := name
			it.ReporterName = &n
		}
		if title, ok := prodMap[r.ProductID]; ok {
			t := title
			it.ProductTitle = &t
		}
		items = append(items, it)
	}
	return items, nil
}

// HandleReport replicates ReportService.handle: partial update of status and
// adminRemark (only fields provided in the request). No product side-effect —
// Java does not take the product offline on report resolution.
func (s *Service) HandleReport(ctx context.Context, id int64, req ReportHandleReq) error {
	r, err := s.q.GetReport(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("举报不存在")
		}
		return err
	}
	status := r.Status
	if req.Status != nil {
		status = *req.Status
	}
	adminRemark := r.AdminRemark
	if req.AdminRemark != nil {
		adminRemark = req.AdminRemark
	}
	return s.q.UpdateReport(ctx, gen.UpdateReportParams{
		ID:          id,
		Status:      status,
		AdminRemark: adminRemark,
	})
}

// --- feedback ---

// FeedbackItem mirrors FeedbackDTO.Item (camelCase wire contract).
type FeedbackItem struct {
	ID         int64      `json:"id"`
	UserID     *int64     `json:"userId"`
	UserName   *string    `json:"userName"`
	Category   string     `json:"category"`
	Content    string     `json:"content"`
	Contact    *string    `json:"contact"`
	Status     string     `json:"status"`
	AdminReply *string    `json:"adminReply"`
	CreatedAt  *time.Time `json:"createdAt"`
}

// FeedbackReplyReq mirrors FeedbackDTO.ReplyReq.
type FeedbackReplyReq struct {
	Status     *string `json:"status"`
	AdminReply *string `json:"adminReply"`
}

// ListFeedback replicates FeedbackService.listAll: optional status filter,
// ordered by created_at desc, enriched with userName.
func (s *Service) ListFeedback(ctx context.Context, status *string) ([]FeedbackItem, error) {
	rows, err := s.q.ListAllFeedback(ctx, blankToNil(status))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return make([]FeedbackItem, 0), nil
	}
	userSet := map[int64]struct{}{}
	for _, f := range rows {
		if f.UserID != nil {
			userSet[*f.UserID] = struct{}{}
		}
	}
	userMap := map[int64]string{}
	if len(userSet) > 0 {
		users, err := s.q.ListUsersByIDs(ctx, keys(userSet))
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			userMap[u.ID] = u.Name
		}
	}
	items := make([]FeedbackItem, 0, len(rows))
	for _, f := range rows {
		it := FeedbackItem{
			ID:         f.ID,
			UserID:     f.UserID,
			Category:   f.Category,
			Content:    f.Content,
			Contact:    f.Contact,
			Status:     f.Status,
			AdminReply: f.AdminReply,
			CreatedAt:  timePtr(f.CreatedAt),
		}
		if f.UserID != nil {
			if name, ok := userMap[*f.UserID]; ok {
				n := name
				it.UserName = &n
			}
		}
		items = append(items, it)
	}
	return items, nil
}

// ReplyFeedback replicates FeedbackService.reply: partial update of status and
// adminReply (adminReply is HTML-escaped, matching Java HtmlUtils.htmlEscape).
func (s *Service) ReplyFeedback(ctx context.Context, id int64, req FeedbackReplyReq) error {
	f, err := s.q.GetFeedback(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.Biz("意见不存在")
		}
		return err
	}
	status := f.Status
	if req.Status != nil {
		status = *req.Status
	}
	adminReply := f.AdminReply
	if req.AdminReply != nil {
		escaped := html.EscapeString(*req.AdminReply)
		adminReply = &escaped
	}
	return s.q.UpdateFeedbackReply(ctx, gen.UpdateFeedbackReplyParams{
		ID:         id,
		Status:     status,
		AdminReply: adminReply,
	})
}

// --- announcements ---

// Announcement mirrors AnnouncementDTO.Item (camelCase wire contract).
type Announcement struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Status    string     `json:"status"`
	CreatedBy *int64     `json:"createdBy"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// AnnouncementSaveReq mirrors AnnouncementDTO.SaveReq.
type AnnouncementSaveReq struct {
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Status  *string `json:"status"`
}

func announcementToItem(a gen.Announcement) Announcement {
	return Announcement{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		Status:    a.Status,
		CreatedBy: a.CreatedBy,
		CreatedAt: timePtr(a.CreatedAt),
		UpdatedAt: timePtr(a.UpdatedAt),
	}
}

// ListAnnouncements replicates AnnouncementService.list: all announcements
// ordered by created_at desc.
func (s *Service) ListAnnouncements(ctx context.Context) ([]Announcement, error) {
	rows, err := s.q.ListAllAnnouncements(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Announcement, 0, len(rows))
	for _, r := range rows {
		items = append(items, announcementToItem(r))
	}
	return items, nil
}

// CreateAnnouncement replicates AnnouncementService.create: title/content are
// trimmed + HTML-escaped, status normalized (default ACTIVE), created_by set to
// the current admin, and when the new one is ACTIVE all others are set INACTIVE.
func (s *Service) CreateAnnouncement(ctx context.Context, adminID int64, req AnnouncementSaveReq) (Announcement, error) {
	if err := validateAnnouncement(req); err != nil {
		return Announcement{}, err
	}
	status, err := normalizeAnnouncementStatus(req.Status)
	if err != nil {
		return Announcement{}, err
	}
	// Insert + disableOtherActive must be atomic (matches Java @Transactional):
	// a failure between the two writes could otherwise leave two ACTIVE
	// announcements, breaking the single-active invariant.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Announcement{}, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit
	qtx := s.newAnnTx(tx)

	created, err := qtx.InsertAnnouncement(ctx, gen.InsertAnnouncementParams{
		Title:     cleanAnnouncement(req.Title),
		Content:   cleanAnnouncement(req.Content),
		Status:    status,
		CreatedBy: &adminID,
	})
	if err != nil {
		return Announcement{}, err
	}
	if status == "ACTIVE" {
		if err := qtx.DisableOtherActiveAnnouncements(ctx, created.ID); err != nil {
			return Announcement{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Announcement{}, err
	}
	final, err := s.q.GetAnnouncement(ctx, created.ID)
	if err != nil {
		return Announcement{}, err
	}
	return announcementToItem(final), nil
}

// UpdateAnnouncement replicates AnnouncementService.update.
func (s *Service) UpdateAnnouncement(ctx context.Context, id int64, req AnnouncementSaveReq) (Announcement, error) {
	if err := validateAnnouncement(req); err != nil {
		return Announcement{}, err
	}
	if _, err := s.q.GetAnnouncement(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Announcement{}, httpx.Biz("公告不存在")
		}
		return Announcement{}, err
	}
	status, err := normalizeAnnouncementStatus(req.Status)
	if err != nil {
		return Announcement{}, err
	}
	// Update + disableOtherActive must be atomic (matches Java @Transactional):
	// a failure between the two writes could otherwise leave two ACTIVE
	// announcements, breaking the single-active invariant.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Announcement{}, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit
	qtx := s.newAnnTx(tx)

	if _, err := qtx.UpdateAnnouncement(ctx, gen.UpdateAnnouncementParams{
		ID:      id,
		Title:   cleanAnnouncement(req.Title),
		Content: cleanAnnouncement(req.Content),
		Status:  status,
	}); err != nil {
		return Announcement{}, err
	}
	if status == "ACTIVE" {
		if err := qtx.DisableOtherActiveAnnouncements(ctx, id); err != nil {
			return Announcement{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Announcement{}, err
	}
	final, err := s.q.GetAnnouncement(ctx, id)
	if err != nil {
		return Announcement{}, err
	}
	return announcementToItem(final), nil
}

// DeleteAnnouncement replicates AnnouncementService.delete: hard delete by id.
func (s *Service) DeleteAnnouncement(ctx context.Context, id int64) error {
	return s.q.DeleteAnnouncement(ctx, id)
}

// validateAnnouncement mirrors the @Valid @NotBlank/@Size constraints on
// AnnouncementDTO.SaveReq (title <= 80, content <= 500, both required).
func validateAnnouncement(req AnnouncementSaveReq) error {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return httpx.Biz("公告标题不能为空")
	}
	if len([]rune(title)) > 80 {
		return httpx.Biz("公告标题不能超过 80 字")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return httpx.Biz("公告内容不能为空")
	}
	if len([]rune(content)) > 500 {
		return httpx.Biz("公告内容不能超过 500 字")
	}
	return nil
}

// normalizeAnnouncementStatus mirrors AnnouncementService.normalizeStatus:
// blank -> ACTIVE, else uppercase must be ACTIVE/INACTIVE.
func normalizeAnnouncementStatus(status *string) (string, error) {
	if status == nil || strings.TrimSpace(*status) == "" {
		return "ACTIVE", nil
	}
	normalized := strings.ToUpper(strings.TrimSpace(*status))
	if normalized != "ACTIVE" && normalized != "INACTIVE" {
		return "", httpx.Biz("公告状态不正确")
	}
	return normalized, nil
}

// cleanAnnouncement mirrors AnnouncementService.clean: trim then HTML-escape.
func cleanAnnouncement(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}

// --- shared helpers ---

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
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
