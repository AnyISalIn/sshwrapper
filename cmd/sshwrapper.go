package main

import (
	"flag"
	"fmt"
	"github.com/AnyISalIn/sshwrapper/config"
	"github.com/AnyISalIn/sshwrapper/gateway"
	"github.com/AnyISalIn/sshwrapper/shared"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"log"
	"net"
	"os"
)

var version = "develop"

var (
	listenAddr *string
	configFile *string
	verFlag    *bool
)

var logger = log.New(os.Stdout, "[main] ", shared.LOG_FLAGS)

func init() {
	listenAddr = flag.String("listen-addr", "0.0.0.0:2022", "sshwrapper listen address")
	configFile = flag.String("config", "./example.yaml", "sshwrapper config file")
	verFlag = flag.Bool("version", false, "show sshwrapper version")

	flag.Parse()
}

func main() {
	if *verFlag {
		fmt.Println(version)
		return
	}

	configBytes, err := ioutil.ReadFile(*configFile)
	if err != nil {
		logger.Fatalf("failed to read file: %v", err)
	}

	config := &config.Config{}

	if err := yaml.Unmarshal(configBytes, config); err != nil {
		logger.Fatalf("failed to unmarshal file: %v", err)
	}

	gw, err := gateway.NewGateway(config)
	if err != nil {
		logger.Fatalf("failed to init gateway: %v", err)
	}

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	logger.Printf("SSH Wrapper Listening on %s", *listenAddr)

	if err := gw.Serve(listener); err != nil {
		logger.Fatalf("failed to serve: %v", err)
	}
}
