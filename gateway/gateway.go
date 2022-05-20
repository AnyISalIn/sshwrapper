package gateway

import (
	"bytes"
	"fmt"
	"github.com/AnyISalIn/sshwrapper/config"
	"github.com/AnyISalIn/sshwrapper/handlers"
	"github.com/AnyISalIn/sshwrapper/router"
	"github.com/AnyISalIn/sshwrapper/shared"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sort"
	"strings"
)

var logger = log.New(os.Stdout, "[gateway] ", shared.LOG_FLAGS)

type Gateway struct {
	router       *router.Router
	routesMap    map[string]config.Route
	userMap      map[string]config.User
	hostKeyBytes []byte
}

func NewGateway(cfg *config.Config) (*Gateway, error) {
	ro := router.NewRouter()
	routesMap := make(map[string]config.Route)
	userMap := make(map[string]config.User)
	// register config handler
	for i := range cfg.Routes {
		var route = cfg.Routes[i]
		if route.Handler.Command != nil {
			routesMap[router.GetURLKey(route.URI)] = route
			ro.RegisterHandler(route.URI, func() handlers.Handler {
				return handlers.NewCommandHandler(*route.Handler.Command)
			})
		}
	}

	// register internal handler
	var internalAPIPath = "/apis"
	var apis []string
	for uri, _ := range routesMap {
		apis = append(apis, uri)
	}
	sort.Strings(apis)
	apiHandler := handlers.NewAPIHandler(apis)
	ro.RegisterHandler(internalAPIPath, func() handlers.Handler {
		return apiHandler
	})
	routesMap[internalAPIPath] = config.Route{
		URI:           internalAPIPath,
		AllowAllUsers: true,
	}

	// register config user
	for i := range cfg.Users {
		user := cfg.Users[i]
		userMap[user.Name] = user
	}

	var hostKeyBytes []byte
	if cfg.HostKeyFile == nil {
		logger.Printf("warning: config not contains hostKeyFile, will be use default, it's not safe")
		hostKeyBytes = []byte(defaultHostKey)
	} else {
		var err error
		hostKeyBytes, err = ioutil.ReadFile(*cfg.HostKeyFile)
		if err != nil {
			return nil, err
		}
	}

	return &Gateway{router: ro, routesMap: routesMap, userMap: userMap, hostKeyBytes: hostKeyBytes}, nil
}

func (g *Gateway) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Printf("failed to accept conn: %v", err)
			continue
		}
		go g.HandleConn(conn)
	}
}

func (g *Gateway) HandleConn(conn net.Conn) {
	defer conn.Close()
	config := g.serverConfig()

	scon, sch, sreqs, err := ssh.NewServerConn(conn, &config)
	if err != nil {
		logger.Printf("failed to accept conn: %v", err)
		return
	}

	go ssh.DiscardRequests(sreqs)
	defer scon.Close()

	loginUser, got := g.userMap[scon.User()]
	if !got {
		return
	}

	for ch := range sch {
		if ch.ChannelType() != "session" {
			ch.Reject(ssh.UnknownChannelType, fmt.Sprintf("unsupported channel type %s", ch.ChannelType()))
		}

		newCh, cReqs, err := ch.Accept()
		if err != nil {
			logger.Printf("failed to accept conn: %v", err)
			return
		}

		// saved channel hold requests before exec req
		saved := make(chan *ssh.Request, 1000)
		for req := range cReqs {
			// parse route
			if req.Type == "exec" {
				var msg shared.MsgExec
				if err := ssh.Unmarshal(req.Payload, &msg); err != nil {
					if req.WantReply {
						req.Reply(false, nil)
					}
					log.Printf("[exec] failed to parse 'exec' payload. %s\n", err)
					break
				}
				// TODO: route params
				segs := strings.Split(msg.Command, " ")
				uri := segs[0]

				route, got := g.routesMap[router.GetURLKey(uri)]
				if !got {
					logger.Printf("failed to match route %s", uri)
					return
				}

				if !route.Allow(loginUser) {
					logger.Printf("user %s -> %s forbidden, err: %v", loginUser.Name, uri, err)
					// TODO: err message
					newCh.SendRequest("exit-status", false, ssh.Marshal(shared.MsgExitStatus{Status: uint32(403)}))
					return
				}

				params := router.ExtraParams(uri)
				params["SSHWRAPPER_USERNAME"] = scon.User()

				logger.Printf("dispatch to %s", uri)
				if err := g.router.Dispatch(router.GetURLKey(uri), newCh, saved, params); err != nil {
					logger.Printf("failed to dispatch %s, err: %v", uri, err)
					return
				}
			} else {
				saved <- req
			}
		}
	}
}

func (g *Gateway) serverConfig() ssh.ServerConfig {
	signer, _ := ssh.ParsePrivateKey(g.hostKeyBytes)
	cfg := ssh.ServerConfig{
		Config:       ssh.Config{},
		NoClientAuth: false,
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			connUser := conn.User()
			for userName, user := range g.userMap {
				if connUser == userName {
					if user.Password != nil && *user.Password == string(password) {
						return new(ssh.Permissions), nil
					}
				}
			}
			return nil, fmt.Errorf("invalid password")
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			connUser := conn.User()
			// TODO: optimization
			for userName, user := range g.userMap {
				if connUser == userName {
					var userPubKey ssh.PublicKey
					var err error

					if user.PublicKey != nil {
						userPubKey, _, _, _, err = ssh.ParseAuthorizedKey([]byte(*user.PublicKey))
						if err != nil {
							return nil, err
						}
					}

					if user.PublicKeyPath != nil {
						pkBytes, err := ioutil.ReadFile(*user.PublicKeyPath)
						if err != nil {
							return nil, err
						}
						userPubKey, _, _, _, err = ssh.ParseAuthorizedKey(pkBytes)
						if err != nil {
							return nil, err
						}
					}
					if userPubKey != nil && bytes.Equal(userPubKey.Marshal(), key.Marshal()) {
						return new(ssh.Permissions), nil
					}
				}
			}
			return nil, fmt.Errorf("invalid publickey")
		},
		KeyboardInteractiveCallback: nil,
		AuthLogCallback:             nil,
		ServerVersion:               "",
		BannerCallback:              nil,
		GSSAPIWithMICConfig:         nil,
	}
	cfg.AddHostKey(signer)
	return cfg
}

var defaultHostKey = `
-----BEGIN OPENSSH PRIVATE KEY-----
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
AwQFBg==
-----END OPENSSH PRIVATE KEY-----
`
