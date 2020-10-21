package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/hsmade/web-expose/pkg/client"
)

var (
	remoteUrl   = kingpin.Flag("remote-url", "websocket URL (ws://host:port/WSconnect)").Required().String()
	localServer = kingpin.Flag("local-server", "local host:port to connect to").Required().String()
)

func main() {
	kingpin.Parse()
	logrus.SetLevel(logrus.DebugLevel)
	c := client.Client{
		RemoteUrl:   *remoteUrl,
		LocalServer: *localServer,
	}

	c.Run()
}
