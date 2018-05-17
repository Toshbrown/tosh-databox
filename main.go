package main

import (
	"flag"
	"os"
	"strings"

	databoxClient "github.com/toshbrown/tosh-databox/databoxClient"
)

func main() {

	//TODO parse CMD input for stop and dev mode etc

	DOCKER_API_VERSION := flag.String("API", "1.35", "Docker API version ")
	CMD := flag.String("cmd", "START", "start,stop")
	IP := flag.String("ip", "192.168.1.131", "The external IP to use")
	//DEV              := flag.Bool("dev", false, "Use this to enable dev mode")
	flag.Parse()

	os.Setenv("DOCKER_API_VERSION", *DOCKER_API_VERSION)

	if _, err := os.Stat("./certs"); err != nil {
		os.Mkdir("./certs", 0770)
	}
	if _, err := os.Stat("./slaStore"); err != nil {
		os.Mkdir("./slaStore", 0770)
	}

	databox := databoxClient.NewDataboxClient()

	switch strings.ToUpper(*CMD) {
	case "STOP":
		databox.Stop()
	case "START":
		databox.Stop()
		databox.Start(*IP)
	}

}
