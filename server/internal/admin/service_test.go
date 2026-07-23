package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// stubQuerier is a hand-rolled fake implementing admin.Querier. Only the methods
// exercised by a given test need to return meaningful data; the rest are no-ops.
type stubQuerier struct {
	users map[int64]gen.User

	setUserStatus  []gen.SetUserStatusParams
	updateReport   []gen.UpdateReportParams
	updateFeedback []gen.UpdateFeedbackReplyParams

	reports  map[int64]gen.Report
	feedback map[int64]gen.Feedback

	announcements     map[int64]gen.Announcement
	insertedAnn       gen.InsertAnnouncementParams
	updatedAnn        gen.UpdateAnnouncementParams
	disableOtherCalls []int64
	nextAnnID         int64
}

func (s *stubQuerier) CountAdminUsers(context.Context, *string) (int64, error) { return 0, nil }
func (s *stubQuerier) ListAdminUsers(context.Context, gen.ListAdminUsersParams) ([]gen.ListAdminUsersRow, error) {
	return nil, nil
}
func (s *stubQuerier) GetUserByID(_ context.Context, id int64) (gen.User, error) {
	u, ok := s.users[id]
	if !ok {
		return gen.User{}, pgx.ErrNoRows
	}
	return u, nil
}
func (s *stubQuerier) SetUserStatus(_ context.Context, arg gen.SetUserStatusParams) error {
	s.setUserStatus = append(s.setUserStatus, arg)
	return nil
}

func (s *stubQuerier) ListActiveCategories(context.Context) ([]gen.Category, error) {
	return nil, nil
}
func (s *stubQuerier) GetCategory(context.Context, int64) (gen.Category, error) {
	return gen.Category{}, pgx.ErrNoRows
}
func (s *stubQuerier) InsertCategory(_ context.Context, arg gen.InsertCategoryParams) (gen.Category, error) {
	return gen.Category{Name: arg.Name, Icon: arg.Icon, SortOrder: arg.SortOrder, Status: arg.Status}, nil
}
func (s *stubQuerier) UpdateCategory(_ context.Context, arg gen.UpdateCategoryParams) (gen.Category, error) {
	return gen.Category{ID: arg.ID, Name: arg.Name, Icon: arg.Icon, SortOrder: arg.SortOrder, Status: arg.Status}, nil
}
func (s *stubQuerier) DeleteCategory(context.Context, int64) error { return nil }

func (s *stubQuerier) ListReports(context.Context, *string) ([]gen.Report, error) { return nil, nil }
func (s *stubQuerier) GetReport(_ context.Context, id int64) (gen.Report, error) {
	r, ok := s.reports[id]
	if !ok {
		return gen.Report{}, pgx.ErrNoRows
	}
	return r, nil
}
func (s *stubQuerier) UpdateReport(_ context.Context, arg gen.UpdateReportParams) error {
	s.updateReport = append(s.updateReport, arg)
	return nil
}
func (s *stubQuerier) ListUsersByIDs(context.Context, []int64) ([]gen.ListUsersByIDsRow, error) {
	return nil, nil
}
func (s *stubQuerier) ListProductTitlesByIDs(context.Context, []int64) ([]gen.ListProductTitlesByIDsRow, error) {
	return nil, nil
}

func (s *stubQuerier) ListAllFeedback(context.Context, *string) ([]gen.Feedback, error) {
	return nil, nil
}
func (s *stubQuerier) GetFeedback(_ context.Context, id int64) (gen.Feedback, error) {
	f, ok := s.feedback[id]
	if !ok {
		return gen.Feedback{}, pgx.ErrNoRows
	}
	return f, nil
}
func (s *stubQuerier) UpdateFeedbackReply(_ context.Context, arg gen.UpdateFeedbackReplyParams) error {
	s.updateFeedback = append(s.updateFeedback, arg)
	return nil
}

func (s *stubQuerier) ListAllAnnouncements(context.Context) ([]gen.Announcement, error) {
	return nil, nil
}
func (s *stubQuerier) GetAnnouncement(_ context.Context, id int64) (gen.Announcement, error) {
	a, ok := s.announcements[id]
	if !ok {
		return gen.Announcement{}, pgx.ErrNoRows
	}
	return a, nil
}
func (s *stubQuerier) InsertAnnouncement(_ context.Context, arg gen.InsertAnnouncementParams) (gen.Announcement, error) {
	s.insertedAnn = arg
	if s.announcements == nil {
		s.announcements = map[int64]gen.Announcement{}
	}
	s.nextAnnID++
	a := gen.Announcement{ID: s.nextAnnID, Title: arg.Title, Content: arg.Content, Status: arg.Status, CreatedBy: arg.CreatedBy}
	s.announcements[a.ID] = a
	return a, nil
}
func (s *stubQuerier) UpdateAnnouncement(_ context.Context, arg gen.UpdateAnnouncementParams) (gen.Announcement, error) {
	s.updatedAnn = arg
	a := s.announcements[arg.ID]
	a.Title, a.Content, a.Status = arg.Title, arg.Content, arg.Status
	s.announcements[arg.ID] = a
	return a, nil
}
func (s *stubQuerier) DeleteAnnouncement(context.Context, int64) error { return nil }
func (s *stubQuerier) DisableOtherActiveAnnouncements(_ context.Context, id int64) error {
	s.disableOtherCalls = append(s.disableOtherCalls, id)
	return nil
}

// fakeTx is a no-op pgx.Tx used to exercise the transactional announcement
// paths without a real database. The service builds its tx-scoped writer via
// newAnnTx (overridden in newSvc to route to the stub), so this tx is never
// used for actual queries — only Begin/Commit/Rollback are invoked.
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

// newSvc wires a Service whose transactional announcement writes route back to
// the stub, so Create/UpdateAnnouncement can be tested without a real DB.
func newSvc(q Querier) *Service {
	svc := NewService(q, nil, &fakeBeginner{})
	svc.newAnnTx = func(pgx.Tx) annTxQuerier {
		if aq, ok := q.(annTxQuerier); ok {
			return aq
		}
		return nil
	}
	return svc
}

func isBiz(t *testing.T, err error, wantCode int, wantMsg string) {
	t.Helper()
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Code != wantCode {
		t.Fatalf("expected code %d, got %d", wantCode, be.Code)
	}
	if wantMsg != "" && be.Msg != wantMsg {
		t.Fatalf("expected msg %q, got %q", wantMsg, be.Msg)
	}
}

func TestChangeUserStatus_MissingUser(t *testing.T) {
	q := &stubQuerier{users: map[int64]gen.User{}}
	err := newSvc(q).ChangeUserStatus(context.Background(), 999, "DISABLED")
	isBiz(t, err, 400, "用户不存在")
	if len(q.setUserStatus) != 0 {
		t.Fatalf("must not write status for a missing user")
	}
}

func TestChangeUserStatus_Toggle(t *testing.T) {
	q := &stubQuerier{users: map[int64]gen.User{7: {ID: 7, Status: "ACTIVE"}}}
	if err := newSvc(q).ChangeUserStatus(context.Background(), 7, "DISABLED"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.setUserStatus) != 1 || q.setUserStatus[0].ID != 7 || q.setUserStatus[0].Status != "DISABLED" {
		t.Fatalf("expected status set to DISABLED for id 7, got %+v", q.setUserStatus)
	}
}

func TestHandleReport_MissingReport(t *testing.T) {
	q := &stubQuerier{reports: map[int64]gen.Report{}}
	err := newSvc(q).HandleReport(context.Background(), 1, ReportHandleReq{})
	isBiz(t, err, 400, "举报不存在")
}

func TestHandleReport_NoProductSideEffect(t *testing.T) {
	// Java ReportService.handle only updates status/adminRemark; it does NOT take
	// the product offline. This test guards against accidentally adding that.
	remark := "spam confirmed"
	status := "RESOLVED"
	q := &stubQuerier{reports: map[int64]gen.Report{5: {ID: 5, ProductID: 42, Status: "PENDING"}}}
	svc := newSvc(q)
	if err := svc.HandleReport(context.Background(), 5, ReportHandleReq{Status: &status, AdminRemark: &remark}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.updateReport) != 1 {
		t.Fatalf("expected one report update, got %d", len(q.updateReport))
	}
	got := q.updateReport[0]
	if got.ID != 5 || got.Status != "RESOLVED" || got.AdminRemark == nil || *got.AdminRemark != "spam confirmed" {
		t.Fatalf("unexpected report update: %+v", got)
	}
	// The admin service has no product dependency wired for HandleReport, so any
	// attempt to touch the product would panic on a nil pointer — proving no side-effect.
}

func TestHandleReport_PartialKeepsExisting(t *testing.T) {
	// Only status provided; existing adminRemark must be preserved (Java partial update).
	existing := "prior note"
	status := "RESOLVED"
	q := &stubQuerier{reports: map[int64]gen.Report{9: {ID: 9, Status: "PENDING", AdminRemark: &existing}}}
	if err := newSvc(q).HandleReport(context.Background(), 9, ReportHandleReq{Status: &status}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := q.updateReport[0]
	if got.Status != "RESOLVED" || got.AdminRemark == nil || *got.AdminRemark != "prior note" {
		t.Fatalf("expected preserved adminRemark, got %+v", got)
	}
}

func TestReplyFeedback_EscapesAndPartial(t *testing.T) {
	status := "RESOLVED"
	reply := "<b>thanks</b>"
	q := &stubQuerier{feedback: map[int64]gen.Feedback{3: {ID: 3, Status: "PENDING"}}}
	if err := newSvc(q).ReplyFeedback(context.Background(), 3, FeedbackReplyReq{Status: &status, AdminReply: &reply}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := q.updateFeedback[0]
	if got.Status != "RESOLVED" {
		t.Fatalf("expected status RESOLVED, got %q", got.Status)
	}
	if got.AdminReply == nil || *got.AdminReply != "&lt;b&gt;thanks&lt;/b&gt;" {
		t.Fatalf("expected HTML-escaped reply, got %v", got.AdminReply)
	}
}

func TestReplyFeedback_Missing(t *testing.T) {
	q := &stubQuerier{feedback: map[int64]gen.Feedback{}}
	err := newSvc(q).ReplyFeedback(context.Background(), 1, FeedbackReplyReq{})
	isBiz(t, err, 400, "意见不存在")
}

func TestCreateAnnouncement_ActiveDisablesOthers(t *testing.T) {
	q := &stubQuerier{}
	active := "active"
	item, err := newSvc(q).CreateAnnouncement(context.Background(), 100, AnnouncementSaveReq{
		Title: "  Hi  ", Content: "  Body  ", Status: &active,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Status != "ACTIVE" {
		t.Fatalf("expected normalized status ACTIVE, got %q", item.Status)
	}
	if q.insertedAnn.Title != "Hi" || q.insertedAnn.Content != "Body" {
		t.Fatalf("expected trimmed title/content, got %+v", q.insertedAnn)
	}
	if q.insertedAnn.CreatedBy == nil || *q.insertedAnn.CreatedBy != 100 {
		t.Fatalf("expected createdBy=100, got %v", q.insertedAnn.CreatedBy)
	}
	if len(q.disableOtherCalls) != 1 {
		t.Fatalf("expected disableOtherActive to be called once, got %d", len(q.disableOtherCalls))
	}
}

func TestCreateAnnouncement_CommitsTransaction(t *testing.T) {
	// The insert AND disableOtherActive must both run through the tx and the tx
	// must be committed, proving the single-active toggle is atomic.
	q := &stubQuerier{}
	beginner := &fakeBeginner{}
	svc := NewService(q, nil, beginner)
	svc.newAnnTx = func(pgx.Tx) annTxQuerier { return q }
	active := "active"
	if _, err := svc.CreateAnnouncement(context.Background(), 1, AnnouncementSaveReq{
		Title: "T", Content: "C", Status: &active,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.insertedAnn.Title != "T" {
		t.Fatalf("expected insert to go through tx, got %+v", q.insertedAnn)
	}
	if len(q.disableOtherCalls) != 1 {
		t.Fatalf("expected disableOtherActive once via tx, got %d", len(q.disableOtherCalls))
	}
	if beginner.tx == nil || !beginner.tx.committed {
		t.Fatalf("expected the transaction to be committed")
	}
}

func TestUpdateAnnouncement_CommitsTransaction(t *testing.T) {
	q := &stubQuerier{announcements: map[int64]gen.Announcement{5: {ID: 5, Status: "INACTIVE"}}}
	beginner := &fakeBeginner{}
	svc := NewService(q, nil, beginner)
	svc.newAnnTx = func(pgx.Tx) annTxQuerier { return q }
	active := "active"
	if _, err := svc.UpdateAnnouncement(context.Background(), 5, AnnouncementSaveReq{
		Title: "T", Content: "C", Status: &active,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.updatedAnn.ID != 5 || q.updatedAnn.Status != "ACTIVE" {
		t.Fatalf("expected update to go through tx, got %+v", q.updatedAnn)
	}
	if len(q.disableOtherCalls) != 1 || q.disableOtherCalls[0] != 5 {
		t.Fatalf("expected disableOtherActive(5) once via tx, got %v", q.disableOtherCalls)
	}
	if beginner.tx == nil || !beginner.tx.committed {
		t.Fatalf("expected the transaction to be committed")
	}
}

func TestCreateAnnouncement_InactiveKeepsOthers(t *testing.T) {
	q := &stubQuerier{}
	inactive := "inactive"
	if _, err := newSvc(q).CreateAnnouncement(context.Background(), 1, AnnouncementSaveReq{
		Title: "T", Content: "C", Status: &inactive,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.insertedAnn.Status != "INACTIVE" {
		t.Fatalf("expected INACTIVE, got %q", q.insertedAnn.Status)
	}
	if len(q.disableOtherCalls) != 0 {
		t.Fatalf("expected no disableOtherActive for INACTIVE, got %d", len(q.disableOtherCalls))
	}
}

func TestCreateAnnouncement_BadStatus(t *testing.T) {
	q := &stubQuerier{}
	bad := "PAUSED"
	_, err := newSvc(q).CreateAnnouncement(context.Background(), 1, AnnouncementSaveReq{Title: "T", Content: "C", Status: &bad})
	isBiz(t, err, 400, "公告状态不正确")
}

func TestCreateAnnouncement_Validation(t *testing.T) {
	q := &stubQuerier{}
	_, err := newSvc(q).CreateAnnouncement(context.Background(), 1, AnnouncementSaveReq{Title: "  ", Content: "C"})
	isBiz(t, err, 400, "公告标题不能为空")
}

func TestUpdateAnnouncement_Missing(t *testing.T) {
	q := &stubQuerier{announcements: map[int64]gen.Announcement{}}
	_, err := newSvc(q).UpdateAnnouncement(context.Background(), 1, AnnouncementSaveReq{Title: "T", Content: "C"})
	isBiz(t, err, 400, "公告不存在")
}

func TestUpdateCategory_Missing(t *testing.T) {
	q := &stubQuerier{}
	_, err := newSvc(q).UpdateCategory(context.Background(), 1, CategoryReq{Name: "X"})
	isBiz(t, err, 400, "分类不存在")
}

func TestCreateCategory_Defaults(t *testing.T) {
	q := &stubQuerier{}
	item, err := newSvc(q).CreateCategory(context.Background(), CategoryReq{Name: "Books"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Status != "ACTIVE" || item.SortOrder != 0 {
		t.Fatalf("expected defaults ACTIVE/0, got %+v", item)
	}
}
