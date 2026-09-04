// Package inputstats monitors keyboard and mouse activity and emits
// interval statistics. Counts only — no key identities or pointer positions.
//
// Call Start to begin monitoring; cancel the context to stop. The returned
// channel is closed when aggregation ends.
package inputstats
