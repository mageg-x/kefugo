package models

import "testing"

func TestSessionMarkFollowUpKeepsFollowStatusAfterRead(t *testing.T) {
	s := &Session{
		SID: "s:v:a:0000000001",
	}

	s.AssignAgent("agent-1", 100)
	s.OnVisitorMessage(110, "m:v:a:0000000001:0000000001")
	s.OnAgentReply(120, "m:v:a:0000000001:0000000002")
	s.MarkRead(121)
	s.MarkFollowUp()

	if got := s.Status(); got != SessionStatusFollowUP {
		t.Fatalf("expected follow status after mark follow-up, got %s", got)
	}
	if s.LastAgentReadTime != 120 && s.LastAgentReadTime != 121 {
		t.Fatalf("expected last agent read time to remain set, got %d", s.LastAgentReadTime)
	}
}

func TestSessionFollowUpDoesNotHideLaterUnreadVisitorMessage(t *testing.T) {
	s := &Session{
		SID: "s:v:a:0000000001",
	}

	s.AssignAgent("agent-1", 100)
	s.OnVisitorMessage(110, "m:v:a:0000000001:0000000001")
	s.OnAgentReply(120, "m:v:a:0000000001:0000000002")
	s.MarkFollowUp()
	s.OnVisitorMessage(130, "m:v:a:0000000001:0000000003")

	if got := s.Status(); got != SessionStatusUnRead {
		t.Fatalf("expected unread status after later visitor message, got %s", got)
	}
}
