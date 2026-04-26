package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kefu-server/models"
	"kefu-server/store"
)

func prepareMessageTestStore(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	kvPath := filepath.Join(base, "badger")
	if _, err := store.InitStore(kvPath); err != nil {
		t.Fatalf("init store failed: %v", err)
	}
	instMessage = nil
	return base
}

func resetStore() {
	if store.KV != nil {
		_ = store.KV.Close()
		store.KV = nil
	}
}

// TestGetMessagesBySessionBefore 验证按游标分页查询历史消息的边界与顺序行为。
func TestGetMessagesBySessionBefore(t *testing.T) {
	base := prepareMessageTestStore(t)
	defer func() {
		resetStore()
		_ = os.RemoveAll(base)
	}()

	ms := GetMsgService()
	if ms == nil {
		t.Fatalf("msg service nil")
	}

	visitorID := "visitorA"
	appID := "appA"
	var msgIDs []string
	for i := 0; i < 5; i++ {
		msg := &models.Message{
			MsgType:   models.WSContentTypeText,
			Content:   "msg-" + string(rune('0'+i)),
			Timestamp: time.Now().Unix() + int64(i),
		}
		msgID, err := ms.SaveMessage(visitorID, appID, 1, msg)
		if err != nil {
			t.Fatalf("save message failed: %v", err)
		}
		msgIDs = append(msgIDs, msgID)
	}

	sid := models.GetSessionID(visitorID, appID, 1)
	rows, err := ms.GetMessagesBySessionBefore(sid, "", 3)
	if err != nil {
		t.Fatalf("GetMessagesBySessionBefore failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].MsgID != msgIDs[2] || rows[2].MsgID != msgIDs[4] {
		t.Fatalf("unexpected order, got %s ... %s", rows[0].MsgID, rows[2].MsgID)
	}

	rows2, err := ms.GetMessagesBySessionBefore(sid, msgIDs[3], 10)
	if err != nil {
		t.Fatalf("GetMessagesBySessionBefore with cursor failed: %v", err)
	}
	if len(rows2) != 3 {
		t.Fatalf("expected 3 rows before cursor (exclusive), got %d", len(rows2))
	}
	if rows2[0].MsgID != msgIDs[0] || rows2[2].MsgID != msgIDs[2] {
		t.Fatalf("unexpected cursor result tail: %s ... %s", rows2[0].MsgID, rows2[2].MsgID)
	}
}
