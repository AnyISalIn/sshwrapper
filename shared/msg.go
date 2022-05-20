package shared

import "golang.org/x/crypto/ssh"

type MsgExitStatus struct {
	Status uint32
}

type MsgExitSignal struct {
	Signal       ssh.Signal
	CoreDumped   bool
	ErrorMessage string
	LanguageTag  string
}

type MsgPtyReq struct {
	Term   string
	Cols   uint32
	Rows   uint32
	Width  uint32
	Height uint32
	Mode   string
}

type MsgWindowChange struct {
	Cols   uint32
	Rows   uint32
	Width  uint32
	Height uint32
}

type MsgExec struct {
	Command string
}

type MsgEnv struct {
	Key   string
	Value string
}

type MsgSignal struct {
	Signal ssh.Signal
}

type MsgForwarding struct {
	DestAddr   string
	DestPort   uint32
	OriginAddr string
	OriginPort uint32
}

type MsgRemoteForward struct {
	BindAddr string
	BindPort uint32
}

type MsgRemoteForwardSuccess struct {
	BindPort uint32
}
