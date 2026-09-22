package api

import "encoding/json"

// alertRules is the per-target rules blob stored in monitors.alerts /
// domains.alerts. Zero values keep pre-rules behavior: first failure alerts,
// no latency rule, cert threshold = alertDaysBefore, unmuted.
//
//	consecutiveFailures — monitor must fail N checks in a row before it flips down
//	latencyWarnMs       — notify once when an up check's ping crosses above this
//	certDays            — domain TLS-expiry threshold (overrides alertDaysBefore)
//	mute                — suppress notifications; incidents/status still record
type alertRules struct {
	ConsecutiveFailures int  `json:"consecutiveFailures"`
	LatencyWarnMs       int  `json:"latencyWarnMs"`
	CertDays            int  `json:"certDays"`
	Mute                bool `json:"mute"`
}

func parseAlertRules(raw json.RawMessage) alertRules {
	var r alertRules
	_ = json.Unmarshal(raw, &r)
	return r
}

// consecutiveDowns counts failed heartbeats, newest-first, stopping at the
// first up. The just-written heartbeat is included.
func (s *Server) consecutiveDowns(monitorID string, limit int) int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM (
		SELECT status FROM heartbeats WHERE monitor_id = ?
		ORDER BY checked_at DESC, rowid DESC LIMIT ?)
		WHERE status = 'down'`, monitorID, limit).Scan(&n)
	return n
}

// prevPing returns the ping of the heartbeat before the latest, or -1.
func (s *Server) prevPing(monitorID string) int {
	var p int
	if err := s.db.QueryRow(`SELECT ping_ms FROM heartbeats
		WHERE monitor_id = ? ORDER BY checked_at DESC, rowid DESC
		LIMIT 1 OFFSET 1`, monitorID).Scan(&p); err != nil {
		return -1
	}
	return p
}
