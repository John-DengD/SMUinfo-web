package message

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// stubQuerier implements Querier for unit tests without a real DB.
type stubQuerier struct {
	users           map[int64]gen.GetUserByIDForMessageRow
	messages        []gen.Message
	insertedMessage gen.InsertMessageParams
	insertCalled    bool
	markedRead      bool
	unreadCount     int64
	productTitles   map[int64]string
}

func (s *stubQuerier) GetUserByIDForMessage(_ context.Context, id int64) (gen.GetUserByIDForMessageRow, error) {
	u, ok := s.users[id]
	if !ok {
		return gen.GetUserByIDForMessageRow{}, pgx.ErrNoRows
	}
	return u, nil
}

func (s *stubQuerier) InsertMessage(_ context.Context, arg gen.InsertMessageParams) (gen.Message, error) {
	s.insertedMessage = arg
	s.insertCalled = true
	return gen.Message{
		ID:         1,
		SenderID:   arg.SenderID,
		ReceiverID: arg.ReceiverID,
		ProductID:  arg.ProductID,
		Content:    arg.Content,
		IsRead:     false,
		CreatedAt:  pgtype.Timestamptz{},
	}, nil
}

func (s *stubQuerier) ListMessagesByUser(_ context.Context, _ int64) ([]gen.Message, error) {
	return s.messages, nil
}

func (s *stubQuerier) ListMessagesBetween(_ context.Context, _ gen.ListMessagesBetweenParams) ([]gen.Message, error) {
	return s.messages, nil
}

func (s *stubQuerier) MarkMessagesRead(_ context.Context, _ gen.MarkMessagesReadParams) error {
	s.markedRead = true
	return nil
}

func (s *stubQuerier) CountUnreadMessages(_ context.Context, _ int64) (int64, error) {
	return s.unreadCount, nil
}

func (s *stubQuerier) GetProductTitleForMessage(_ context.Context, id int64) (string, error) {
	if s.productTitles != nil {
		if t, ok := s.productTitles[id]; ok {
			return t, nil
		}
	}
	return "", pgx.ErrNoRows
}

// --- Send guard tests ---

func TestSendSelfMessageReturnsError(t *testing.T) {
	svc := NewService(&stubQuerier{})
	receiverID := int64(42)
	_, err := svc.Send(context.Background(), 42, SendReq{ReceiverID: &receiverID, Content: "hello"})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "不能给自己发消息" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestSendNilReceiverReturnsError(t *testing.T) {
	svc := NewService(&stubQuerier{})
	_, err := svc.Send(context.Background(), 1, SendReq{ReceiverID: nil, Content: "hello"})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "参数错误" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestSendEmptyContentReturnsError(t *testing.T) {
	svc := NewService(&stubQuerier{})
	receiverID := int64(2)
	_, err := svc.Send(context.Background(), 1, SendReq{ReceiverID: &receiverID, Content: "   "})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "参数错误" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestSendReceiverNotFoundReturnsError(t *testing.T) {
	svc := NewService(&stubQuerier{users: map[int64]gen.GetUserByIDForMessageRow{}})
	receiverID := int64(999)
	_, err := svc.Send(context.Background(), 1, SendReq{ReceiverID: &receiverID, Content: "hello"})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "接收人不存在" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}

func TestSendHtmlEscapesContent(t *testing.T) {
	stub := &stubQuerier{
		users: map[int64]gen.GetUserByIDForMessageRow{
			2: {ID: 2, Name: "Bob"},
		},
	}
	svc := NewService(stub)
	receiverID := int64(2)
	_, err := svc.Send(context.Background(), 1, SendReq{ReceiverID: &receiverID, Content: "<script>alert(1)</script>"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.insertCalled {
		t.Fatal("expected InsertMessage to be called")
	}
	if stub.insertedMessage.Content != "&lt;script&gt;alert(1)&lt;/script&gt;" {
		t.Fatalf("expected html-escaped content, got %q", stub.insertedMessage.Content)
	}
}

func TestSendSuccessStoresCorrectParams(t *testing.T) {
	stub := &stubQuerier{
		users: map[int64]gen.GetUserByIDForMessageRow{
			5: {ID: 5, Name: "Alice"},
		},
	}
	svc := NewService(stub)
	receiverID := int64(5)
	item, err := svc.Send(context.Background(), 3, SendReq{ReceiverID: &receiverID, Content: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.SenderID != 3 || item.ReceiverID != 5 {
		t.Fatalf("unexpected sender/receiver: %d/%d", item.SenderID, item.ReceiverID)
	}
	if stub.insertedMessage.SenderID != 3 || stub.insertedMessage.ReceiverID != 5 {
		t.Fatalf("InsertMessage called with wrong params: %+v", stub.insertedMessage)
	}
}

// --- Conversations grouping tests ---

func TestConversationsEmptyReturnsEmptySlice(t *testing.T) {
	svc := NewService(&stubQuerier{messages: []gen.Message{}})
	convs, err := svc.Conversations(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if convs == nil || len(convs) != 0 {
		t.Fatalf("expected empty slice, got %v", convs)
	}
}

func TestConversationsGroupsByPeer(t *testing.T) {
	// User 1 exchanged 2 messages with user 2, and 1 with user 3.
	stub := &stubQuerier{
		messages: []gen.Message{
			{ID: 3, SenderID: 3, ReceiverID: 1, IsRead: false, Content: "hey", CreatedAt: pgtype.Timestamptz{}},
			{ID: 2, SenderID: 1, ReceiverID: 2, IsRead: true, Content: "bye", CreatedAt: pgtype.Timestamptz{}},
			{ID: 1, SenderID: 2, ReceiverID: 1, IsRead: false, Content: "hi", CreatedAt: pgtype.Timestamptz{}},
		},
		users: map[int64]gen.GetUserByIDForMessageRow{
			2: {ID: 2, Name: "Bob"},
			3: {ID: 3, Name: "Carol"},
		},
	}
	svc := NewService(stub)
	convs, err := svc.Conversations(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}
	// First conversation should be with peer 3 (newest message first)
	if convs[0].PeerID != 3 {
		t.Fatalf("expected first peer=3, got %d", convs[0].PeerID)
	}
	if convs[0].UnreadCount != 1 {
		t.Fatalf("expected unread=1 for peer 3, got %d", convs[0].UnreadCount)
	}
	// Second conversation with peer 2: unread sent to us (msg id=1) is 1
	if convs[1].PeerID != 2 {
		t.Fatalf("expected second peer=2, got %d", convs[1].PeerID)
	}
	if convs[1].UnreadCount != 1 {
		t.Fatalf("expected unread=1 for peer 2, got %d", convs[1].UnreadCount)
	}
}

// --- UnreadCount test ---

func TestUnreadCountReturnsCorrectValue(t *testing.T) {
	stub := &stubQuerier{unreadCount: 7}
	svc := NewService(stub)
	count, err := svc.UnreadCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Fatalf("expected 7, got %d", count)
	}
}

// --- Conversation (thread) mark-read test ---

func TestConversationMarksUnreadAsRead(t *testing.T) {
	// Message from peer (2) to us (1) is unread - should trigger MarkMessagesRead.
	// The returned items must also reflect IsRead=true so the wire response is consistent
	// with the DB state (guards against returning stale IsRead=false after the bulk update).
	stub := &stubQuerier{
		messages: []gen.Message{
			{ID: 1, SenderID: 2, ReceiverID: 1, IsRead: false, Content: "hi", CreatedAt: pgtype.Timestamptz{}},
			// A message we sent — should remain IsRead=false in the response (we are not the receiver).
			{ID: 2, SenderID: 1, ReceiverID: 2, IsRead: false, Content: "hey", CreatedAt: pgtype.Timestamptz{}},
		},
		users: map[int64]gen.GetUserByIDForMessageRow{
			2: {ID: 2, Name: "Bob"},
		},
	}
	svc := NewService(stub)
	items, err := svc.Conversation(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.markedRead {
		t.Fatal("expected MarkMessagesRead to be called")
	}
	// Assert that the returned item where we (userID=1) are the receiver is now IsRead=true.
	// This fails if the service forgets to update the in-memory slice before serialising.
	for _, it := range items {
		if it.ReceiverID == 1 && !it.IsRead {
			t.Fatalf("item id=%d has ReceiverID=1 but IsRead=false — stale read state returned to caller", it.ID)
		}
	}
	// The message we sent (SenderID=1) should not have its IsRead mutated to true.
	for _, it := range items {
		if it.SenderID == 1 && it.ReceiverID == 2 {
			if it.IsRead {
				t.Fatalf("item id=%d is a sent message but IsRead was unexpectedly set to true", it.ID)
			}
		}
	}
}

func TestConversationEmptyReturnsEmptySlice(t *testing.T) {
	svc := NewService(&stubQuerier{messages: []gen.Message{}})
	items, err := svc.Conversation(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("expected empty slice, got %v", items)
	}
}
