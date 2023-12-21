package main

import (
	"fmt"
	"github.com/akmalfairuz/raknet-proxy/proxy"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <proxy address> <downstream address>\n", os.Args[0])
		fmt.Printf("Example: %s 0.0.0.0:19132 play.venitymc.com:19132\n", os.Args[0])
		os.Exit(1)
		return
	}

	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	proxyAddress := os.Args[1]
	downstreamAddress := os.Args[2]

	prox := proxy.New(log, proxyAddress, downstreamAddress)
	prox.Start()
}
