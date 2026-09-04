package linux

// mouseState tracks REL motion awaiting SYN_REPORT coalescing.
type mouseState struct {
	pendingMove bool
}
