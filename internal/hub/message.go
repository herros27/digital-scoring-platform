package hub

import "strings"

type MessageType string

const (
	MsgScoreUpdate    MessageType = "score_update"
	MsgScoreConfirmed MessageType = "score_confirmed"
	MsgJudgeInput     MessageType = "judge_input"
	MsgStateChange    MessageType = "state_change"
	MsgTimerSync      MessageType = "timer_sync"
	// MsgConsensusProgress: kemajuan kesepakatan juri (jumlah juri yang sudah menekan hal yang sama), hanya ke tablet juri
	MsgConsensusProgress MessageType = "consensus_progress"
	MsgMatchResult       MessageType = "match_result"
	MsgPenalty           MessageType = "penalty"
	MsgError             MessageType = "error"
	MsgSyncRequest       MessageType = "sync_request"
	MsgSyncResponse      MessageType = "sync_response"

	// Meja wasit: pembatalan sanksi, daftar sanksi, sinyal arena, kehadiran perangkat, log kejadian
	MsgPenaltyRevoke MessageType = "penalty_revoke"
	MsgPenaltyUpdate MessageType = "penalties"
	MsgArenaSignal   MessageType = "arena_signal"
	MsgPresence      MessageType = "presence"
	MsgMatchEvent    MessageType = "match_event"
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

// RoleFromClaim memetakan role pada JWT (auth) ke role hub WebSocket.
// Wasit, dewan, dan referee mengendalikan pertandingan sehingga semuanya menjadi
// RoleReferee; "juri" dinormalkan menjadi RoleJudge. Role lain dipakai apa adanya.
func RoleFromClaim(claimRole string) Role {
	switch claimRole {
	case "wasit", "dewan", "referee":
		return RoleReferee
	case "juri":
		return RoleJudge
	default:
		return Role(claimRole)
	}
}

// IsOperatorClaim true untuk role petugas yang boleh memakai token tanpa arena
// (login email). Sejalan dengan REST: admin, dewan, dan wasit boleh mengontrol
// pertandingan di arena mana pun.
func IsOperatorClaim(claimRole string) bool {
	return claimRole == "admin" || claimRole == "dewan" || claimRole == "wasit"
}
