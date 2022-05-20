package handlers

import (
	"fmt"
	"github.com/AnyISalIn/sshwrapper/shared"
	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"os/exec"
	"time"
)

var logger = log.New(os.Stdout, "[handler] ", shared.LOG_FLAGS)

type NewHandlerFunc func() Handler

type Handler interface {
	InjectParameters(map[string]string)
	Handle(ch ssh.Channel, reqs <-chan *ssh.Request)
}

type APIHandler struct {
	apis []string
}

func NewAPIHandler(apis []string) *APIHandler {
	return &APIHandler{apis: apis}
}

func (a *APIHandler) InjectParameters(m map[string]string) {
	return
}

func (a *APIHandler) Handle(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	ch.Write([]byte(fmt.Sprintf("\n")))
	ch.Write([]byte(fmt.Sprintf("SSH Wrapper API List:\r\n")))
	for _, api := range a.apis {
		ch.Write([]byte(fmt.Sprintf("- %s\r\n", api)))
	}
	ch.Write([]byte(fmt.Sprintf("\n")))
}

type CommandHandler struct {
	cmd     string
	args    []string
	params  map[string]string
	timeout *time.Duration
}

type CommandConfig struct {
	Cmd     string         `yaml:"cmd,omitempty"`
	Args    []string       `yaml:"args,omitempty"`
	Timeout *time.Duration `yaml:"timeout,omitempty"`
}

func NewCommandHandler(config CommandConfig) Handler {
	return &CommandHandler{
		cmd:     config.Cmd,
		args:    config.Args,
		timeout: config.Timeout,
		params:  make(map[string]string),
	}
}

func (c *CommandHandler) InjectParameters(params map[string]string) {
	c.params = params
}

func (c *CommandHandler) Handle(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()

	var envs []string = nil
	var cmd *exec.Cmd = nil
	var ptyFile *os.File = nil
	var ptyReq *shared.MsgPtyReq = nil
	var ptyCh = make(chan struct{})

	// inject env
	for k, v := range c.params {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	envs = append(envs, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))
	if homeDir, err := os.UserHomeDir(); err == nil {
		envs = append(envs, fmt.Sprintf("HOME=%s", homeDir))
	}

	go func() {
		for req := range reqs {
			switch req.Type {
			case "pty-req":
				var ok bool = false
				for {
					if ptyReq != nil {
						logger.Printf("[pty-req] pty has already been created\n")
						break
					}
					msg := &shared.MsgPtyReq{}
					if err := ssh.Unmarshal(req.Payload, msg); err != nil {
						logger.Printf("[pty-req] failed to parse 'pty-req' payload (%s)\n", err)
						break
					}
					ptyReq = msg
					ok = true
					logger.Printf("[pty-req] term:'%s' cols:%d rows:%d\n", msg.Term, msg.Cols, msg.Rows)
					break
				}
				if req.WantReply {
					req.Reply(ok, nil)
				}
				ptyCh <- struct{}{}
			case "window-change":
				var ok bool = false
				for {
					var msg shared.MsgWindowChange
					err := ssh.Unmarshal(req.Payload, &msg)
					if err != nil {
						logger.Printf("[window-change] failed to parse 'window-change' payload (%s)\n", err)
						break
					}

					if ptyFile == nil {
						logger.Printf("[window-change] pty not found\n")
						break
					}

					if err = pty.Setsize(ptyFile, &pty.Winsize{Cols: uint16(msg.Cols), Rows: uint16(msg.Rows), X: uint16(msg.Width), Y: uint16(msg.Height)}); err != nil {
						logger.Printf("[window-change] failed to set pty window size (%s)\n", err)
						break
					}

					logger.Printf("[window-change] cols:%d rows:%d\n", msg.Cols, msg.Rows)
					ok = true
					break
				}
				if req.WantReply {
					req.Reply(ok, nil)
				}
			}
		}
	}()

	<-ptyCh
	cmd = exec.Command(c.cmd, c.args...)
	cmd.Env = envs
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))

	var err error
	if ptyFile, err = pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(ptyReq.Cols), Rows: uint16(ptyReq.Rows), X: uint16(ptyReq.Width), Y: uint16(ptyReq.Height)}); err != nil {
		logger.Print("failed to start pty, err :%v", err)
		return
	}
	defer ptyFile.Close()

	go shared.Bridge(ptyFile, ch)

	cmd.Wait()
	exitCode := cmd.ProcessState.ExitCode()
	ch.SendRequest("exit-status", false, ssh.Marshal(shared.MsgExitStatus{Status: uint32(exitCode)}))
}
