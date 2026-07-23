package message

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

// Querier is the subset of sqlc-generated *gen.Queries the message service needs.
type Querier interface {
	GetUserByIDForMessage(ctx context.Context, id int64) (gen.GetUserByIDForMessageRow, error)
	InsertMessage(ctx context.Context, arg gen.InsertMessageParams) (gen.Message, error)
	ListMessagesByUser(ctx context.Context, userID int64) ([]gen.Message, error)
	ListMessagesBetween(ctx context.Context, arg gen.ListMessagesBetweenParams) ([]gen.Message, error)
	MarkMessagesRead(ctx context.Context, arg gen.MarkMessagesReadParams) error
	CountUnreadMessages(ctx context.Context, receiverID int64) (int64, error)
	GetProductTitleForMessage(ctx context.Context, id int64) (string, error)
}

// Service handles messaging business logic.
type Service struct {
	q Querier
}

// NewService constructs the message service.
func NewService(q Querier) *Service {
	return &Service{q: q}
}

// SendReq mirrors MessageDTO.SendReq.
type SendReq struct {
	ReceiverID *int64  `json:"receiverId"`
	ProductID  *int64  `json:"productId"`
	Content    string  `json:"content"`
}

// Item mirrors MessageDTO.Item (camelCase wire contract).
type Item struct {
	ID           int64      `json:"id"`
	SenderID     int64      `json:"senderId"`
	ReceiverID   int64      `json:"receiverId"`
	SenderName   *string    `json:"senderName"`
	ProductID    *int64     `json:"productId"`
	ProductTitle *string    `json:"productTitle"`
	Content      string     `json:"content"`
	IsRead       bool       `json:"isRead"`
	CreatedAt    *time.Time `json:"createdAt"`
}

// Conversation mirrors MessageDTO.Conversation (camelCase wire contract).
type Conversation struct {
	PeerID       int64      `json:"peerId"`
	PeerName     *string    `json:"peerName"`
	LastContent  string     `json:"lastContent"`
	LastTime     *time.Time `json:"lastTime"`
	UnreadCount  int        `json:"unreadCount"`
	ProductID    *int64     `json:"productId"`
	ProductTitle *string    `json:"productTitle"`
}

// UnreadCountResp mirrors Map.of("count", count) from Java.
type UnreadCountResp struct {
	Count int `json:"count"`
}

// Send replicates MessageService.send:
// - cannot message self
// - receiver must exist
// - content is html-escaped
// - is_read defaults to false
func (s *Service) Send(ctx context.Context, senderID int64, req SendReq) (Item, error) {
	if req.ReceiverID == nil {
		return Item{}, httpx.Biz("参数错误")
	}
	if strings.TrimSpace(req.Content) == "" {
		return Item{}, httpx.Biz("参数错误")
	}
	if senderID == *req.ReceiverID {
		return Item{}, httpx.Biz("不能给自己发消息")
	}
	_, err := s.q.GetUserByIDForMessage(ctx, *req.ReceiverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, httpx.Biz("接收人不存在")
		}
		return Item{}, err
	}

	m, err := s.q.InsertMessage(ctx, gen.InsertMessageParams{
		SenderID:   senderID,
		ReceiverID: *req.ReceiverID,
		ProductID:  req.ProductID,
		Content:    html.EscapeString(req.Content),
	})
	if err != nil {
		return Item{}, err
	}
	return toItem(m), nil
}

// Conversations replicates MessageService.conversations:
// Groups all messages involving the user by counterpart (peer),
// using the first (newest) message per peer as lastContent/lastTime/productId.
// Counts unread messages where the current user is receiver and the message is unread.
// Enriches with peerName and productTitle.
func (s *Service) Conversations(ctx context.Context, userID int64) ([]Conversation, error) {
	all, err := s.q.ListMessagesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// LinkedHashMap equivalent: preserve insertion order (first-seen = latest message)
	type convEntry struct {
		conv  Conversation
		order int
	}
	convMap := map[int64]*Conversation{}
	convOrder := []int64{} // preserves insertion order

	for _, m := range all {
		var peer int64
		if m.SenderID == userID {
			peer = m.ReceiverID
		} else {
			peer = m.SenderID
		}
		c, exists := convMap[peer]
		if !exists {
			// First occurrence = last message (list is newest-first)
			conv := Conversation{
				PeerID:      peer,
				LastContent: m.Content,
				LastTime:    timestampPtr(m.CreatedAt),
				ProductID:   m.ProductID,
				UnreadCount: 0,
			}
			convMap[peer] = &conv
			convOrder = append(convOrder, peer)
			c = convMap[peer]
		}
		// Count unread: messages where we are receiver and not yet read
		if !m.IsRead && m.ReceiverID == userID {
			c.UnreadCount++
		}
	}

	if len(convMap) == 0 {
		return []Conversation{}, nil
	}

	// Batch-load peer users
	peerIDs := convOrder
	userMap := map[int64]gen.GetUserByIDForMessageRow{}
	for _, pid := range peerIDs {
		u, err := s.q.GetUserByIDForMessage(ctx, pid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		userMap[pid] = u
	}

	// Batch-load product titles for unique productIDs
	productTitleMap := map[int64]string{}
	for _, c := range convMap {
		if c.ProductID != nil {
			if _, seen := productTitleMap[*c.ProductID]; !seen {
				title, err := s.q.GetProductTitleForMessage(ctx, *c.ProductID)
				if err == nil {
					productTitleMap[*c.ProductID] = title
				}
				// If product not found, skip (title stays nil)
			}
		}
	}

	// Assemble result in insertion order (newest conversation first)
	result := make([]Conversation, 0, len(convOrder))
	for _, peer := range convOrder {
		c := *convMap[peer]
		if u, ok := userMap[peer]; ok {
			n := u.Name
			c.PeerName = &n
		}
		if c.ProductID != nil {
			if title, ok := productTitleMap[*c.ProductID]; ok {
				t := title
				c.ProductTitle = &t
			}
		}
		result = append(result, c)
	}
	return result, nil
}

// Conversation replicates MessageService.conversation:
// Returns full thread (oldest first) between current user and peer.
// Marks messages sent by peer to current user (unread) as read.
// Enriches with senderName and productTitle.
func (s *Service) Conversation(ctx context.Context, userID, peerID int64) ([]Item, error) {
	msgs, err := s.q.ListMessagesBetween(ctx, gen.ListMessagesBetweenParams{
		SenderID:   userID,
		ReceiverID: peerID,
	})
	if err != nil {
		return nil, err
	}

	// Mark unread messages from peer to us as read (matches Java's loop update)
	for _, m := range msgs {
		if m.ReceiverID == userID && !m.IsRead {
			if err := s.q.MarkMessagesRead(ctx, gen.MarkMessagesReadParams{
				ReceiverID: userID,
				SenderID:   peerID,
			}); err != nil {
				return nil, err
			}
			break // single bulk update is sufficient; break after first call
		}
	}

	if len(msgs) == 0 {
		return []Item{}, nil
	}

	// Collect unique senderIDs for name enrichment
	senderSet := map[int64]struct{}{}
	productSet := map[int64]struct{}{}
	for _, m := range msgs {
		senderSet[m.SenderID] = struct{}{}
		if m.ProductID != nil {
			productSet[*m.ProductID] = struct{}{}
		}
	}

	userMap := map[int64]gen.GetUserByIDForMessageRow{}
	for sid := range senderSet {
		u, err := s.q.GetUserByIDForMessage(ctx, sid)
		if err == nil {
			userMap[sid] = u
		}
	}

	productTitleMap := map[int64]string{}
	for pid := range productSet {
		title, err := s.q.GetProductTitleForMessage(ctx, pid)
		if err == nil {
			productTitleMap[pid] = title
		}
	}

	result := make([]Item, 0, len(msgs))
	for _, m := range msgs {
		it := toItem(m)
		if u, ok := userMap[m.SenderID]; ok {
			n := u.Name
			it.SenderName = &n
		}
		if m.ProductID != nil {
			if title, ok := productTitleMap[*m.ProductID]; ok {
				t := title
				it.ProductTitle = &t
			}
		}
		result = append(result, it)
	}
	return result, nil
}

// UnreadCount replicates MessageService.unreadCount: total unread for the user.
func (s *Service) UnreadCount(ctx context.Context, userID int64) (int, error) {
	c, err := s.q.CountUnreadMessages(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int(c), nil
}

// toItem converts a gen.Message to a wire Item.
func toItem(m gen.Message) Item {
	return Item{
		ID:         m.ID,
		SenderID:   m.SenderID,
		ReceiverID: m.ReceiverID,
		ProductID:  m.ProductID,
		Content:    m.Content,
		IsRead:     m.IsRead,
		CreatedAt:  timestampPtr(m.CreatedAt),
	}
}

func timestampPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
