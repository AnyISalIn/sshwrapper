package router

import (
	"fmt"
	"github.com/AnyISalIn/sshwrapper/handlers"
	"github.com/AnyISalIn/sshwrapper/shared"
	"golang.org/x/crypto/ssh"
	"log"
	"net/url"
	"os"
	"strings"
)

var logger = log.New(os.Stdout, "[router] ", shared.LOG_FLAGS)

type Router struct {
	handlerFuncs map[string]handlers.NewHandlerFunc
}

func NewRouter() *Router {
	return &Router{handlerFuncs: make(map[string]handlers.NewHandlerFunc)}
}

func (r *Router) RegisterHandler(path string, handlerFunc handlers.NewHandlerFunc) {
	r.handlerFuncs[path] = handlerFunc
}

func (r *Router) Dispatch(path string, ch ssh.Channel, reqs <-chan *ssh.Request, params map[string]string) error {
	handlerFunc, got := r.handlerFuncs[path]
	if !got {
		return fmt.Errorf("can't found path %s handlerFunc", path)
	}

	logger.Printf("Handle %s -> %s", path, handlerFunc)
	handler := handlerFunc()
	handler.InjectParameters(params)
	go handler.Handle(ch, reqs)
	return nil
}

func GetURLKey(uri string) string {
	segs := strings.SplitN(uri, "?", 2)
	if len(segs) != 2 {
		return uri
	}
	return segs[0]
}

func ExtraParams(uri string) map[string]string {
	out := make(map[string]string)
	segs := strings.SplitN(uri, "?", 2)
	if len(segs) != 2 {
		return out
	}

	query, err := url.ParseQuery(segs[1])
	if err == nil {
		for k, v := range query {
			out[k] = strings.Join(v, ",")
		}
	}
	return out
}
