package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/hsmade/web-expose/pkg/server"
)

var (
	port = kingpin.Flag("port", "port to listen on for both websocket and web requests").Required().Int()
)

func main() {
	kingpin.Parse()
	logrus.SetLevel(logrus.DebugLevel)
	s := server.NewServer(*port)
	logrus.Fatal(s.Run())
}
