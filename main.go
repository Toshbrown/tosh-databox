package main

import (
	"flag"
	"fmt"
	"os"

	"databoxLoader"
	"databoxLogParser"
	log "databoxlog"
)

func main() {

	//TODO parse CMD input for stop and dev mode etc

	DOCKER_API_VERSION := flag.String("API", "1.35", "Docker API version ")

	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	startCmdIP := startCmd.String("ip", "192.168.0.131", "The external IP to use")
	startCmdRelease := startCmd.String("rel", "0.4.0", "Databox version to start, can uses tagged versions or latest")
	//TODO sort out the cm image name
	cmImage := startCmd.String("cm", "go-container-manager", "Override container-manager image")
	arbiterImage := startCmd.String("arbiter", "databoxsystems/arbiter", "Override arbiter image")
	coreNetworkImage := startCmd.String("core-network", "databoxsystems/core-network", "Override container-manager image")
	coreNetworkRelay := startCmd.String("core-network-relay", "databoxsystems/core-network-relay", "Override core-network-relay image")
	appServerImage := startCmd.String("app-server", "databoxsystems/app-server", "Override local app-server image")
	exportServerImage := startCmd.String("export-service", "databoxsystems/export-service", "Override export-service image")

	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	logsCmd := flag.NewFlagSet("logs", flag.ExitOnError)

	//DEV              := flag.Bool("dev", false, "Use this to enable dev mode")
	flag.Parse()

	os.Setenv("DOCKER_API_VERSION", *DOCKER_API_VERSION)

	if _, err := os.Stat("./certs"); err != nil {
		os.Mkdir("./certs", 0770)
	}
	if _, err := os.Stat("./slaStore"); err != nil {
		os.Mkdir("./slaStore", 0770)
	}

	if len(os.Args) == 1 {
		displayUsage()
		os.Exit(2)
	}

	startCmd.Parse(os.Args[2:])
	databox := databoxLoader.New(*startCmdRelease)

	switch os.Args[1] {
	case "start":
		log.Info("Starting Databox ...")
		databox.Start(*startCmdIP,
			*cmImage,
			*arbiterImage,
			*coreNetworkImage,
			*coreNetworkRelay,
			*appServerImage,
			*exportServerImage)
	case "stop":
		log.Info("Stoping Databox ...")
		stopCmd.Parse(os.Args[2:])
		databox.Stop()
	case "logs":
		logsCmd.Parse(os.Args[2:])
		databoxLogParser.ShowLogs()
	default:
		displayUsage()
		os.Exit(2)
	}

}

func displayUsage() {
	fmt.Println(`
		databox [cmd]
		Usage:
			start - start databox
				Options:
					--ip The external IP to use to advertise the swarm
					--rel Databox version to start, can uses tagged versions or latest
					--cm Override container-manager image
					--arbiter Override arbiter image
					--core-network Override container-manager image
					--core-network-relay Override core-network-relay image
					--app-server Override local app-server image
					--export-service Override export-service image

			stop - stop databox

			logs - view databox logs
		`)
}
