package hub

import "strings"

type MessageType string

const (
	MsgScoreUpdate  MessageType = "score_update"
	MsgJudgeInput   MessageType = "judge_input"
	MsgStateChange  MessageType = "state_change"
	MsgTimerSync    MessageType = "timer_sync"
	MsgMatchResult  MessageType = "match_result"
	MsgPenalty      MessageType = "penalty"
	MsgError        MessageType = "error"
	MsgSyncRequest  MessageType = "sync_request"
	MsgSyncResponse MessageType = "sync_response"
)

type Role string

const (
	RoleJudge      Role = "judge"
	RoleReferee    Role = "referee"
	RoleScoreboard Role = "scoreboard"
	RoleAdmin      Role = "admin"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
	From    string      `json:"from,omitempty"` // judge_id
}

type OutboundMessage struct {
	msg        Message
	targetRole Role // kosong = broadcast ke semua
}

// Tambahkan method ini di package hub
func (r Role) IsJudge() bool {
	return strings.HasPrefix(string(r), "judge")
}
