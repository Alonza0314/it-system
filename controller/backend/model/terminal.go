package model

// Runner terminal protocol, shared by the controller and the runner.
// See design/runner-terminal.md for the full flow.

const (
	// TerminalSubprotocol is the WebSocket subprotocol of every terminal connection.
	TerminalSubprotocol = "it-terminal.v1"
	// TerminalTicketProtocolPrefix marks the one-time ticket that the browser
	// sends as a second subprotocol ("ticket.<ticket>"), because a browser
	// WebSocket cannot send an Authorization header.
	TerminalTicketProtocolPrefix = "ticket."

	TerminalProtocolVersion = 1
)

// Text frame types. Binary frames carry raw terminal bytes in both directions.
const (
	// control channel, controller -> runner
	TerminalMsgOpen = "open"
	// control channel, runner -> controller
	TerminalMsgHello    = "hello"
	TerminalMsgRejected = "rejected"

	// data channel, browser -> runner
	TerminalMsgResize = "resize"
	// data channel, -> browser
	TerminalMsgReady = "ready"
	TerminalMsgExit  = "exit"
	TerminalMsgError = "error"
)

// WebSocket close codes sent to the browser.
const (
	TerminalCloseForbidden        = 4403
	TerminalCloseTimeout          = 4408
	TerminalCloseRunnerGone       = 4409
	TerminalCloseLimitReached     = 4429
	TerminalCloseRunnerNoResponse = 4504
)

type TerminalMessage struct {
	Type string `json:"type"`

	Session string `json:"session,omitempty"`
	Cols    uint16 `json:"cols,omitempty"`
	Rows    uint16 `json:"rows,omitempty"`
	User    string `json:"user,omitempty"`

	Version     int `json:"version,omitempty"`
	MaxSessions int `json:"maxSessions,omitempty"`

	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type RequestTerminalTicket struct {
	Runner string `json:"runner" binding:"required"`
	Cols   uint16 `json:"cols,omitempty"`
	Rows   uint16 `json:"rows,omitempty"`
}

type ResponseTerminalTicket struct {
	Message   string `json:"message" binding:"required"`
	Ticket    string `json:"ticket,omitempty"`
	ExpiresIn int    `json:"expiresIn,omitempty"`
}
