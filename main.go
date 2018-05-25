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
		databox.Start(*startCmdIP)
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
			stop - stop databox
			logs - view databox logs
		`)
}
