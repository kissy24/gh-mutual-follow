package domain

// Error contains only a safe, user-facing message, never raw HTTP or command output.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }
