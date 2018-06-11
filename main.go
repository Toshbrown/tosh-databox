package main

import (
	"flag"
	"fmt"
	"os"

	"databoxLoader"
	"databoxLogParser"
	log "databoxlog"
	databoxtype "lib-go-databox/types"
)

func main() {

	DOCKER_API_VERSION := flag.String("API", "1.35", "Docker API version ")

	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	startCmdIP := startCmd.String("swarm-ip", "127.0.0.1", "The external IP to use")
	startCmdRelease := startCmd.String("release", "0.4.0", "Databox version to start, can uses tagged versions or latest")
	startCmdRegistry := startCmd.String("registry", "databoxsystems", "Override the default registry, where images are pulled form")
	startCmdPassword := startCmd.String("password", "", "Override the password if you dont want an auto generated one. Mainly for testing")
	appStore := startCmd.String("appstore", "https://store.iotdatabox.com", "Override the default appstore where manifests are loaded form")
	//TODO sort out the cm image name
	cmImage := startCmd.String("cm", "go-container-manager", "Override container-manager image")
	arbiterImage := startCmd.String("arbiter", "databoxsystems/arbiter", "Override arbiter image")
	coreNetworkImage := startCmd.String("core-network", "databoxsystems/core-network", "Override container-manager image")
	coreNetworkRelay := startCmd.String("core-network-relay", "databoxsystems/core-network-relay", "Override core-network-relay image")
	appServerImage := startCmd.String("app-server", "databoxsystems/app-server", "Override local app-server image")
	exportServerImage := startCmd.String("export-service", "databoxsystems/export-service", "Override export-service image")
	storeImage := startCmd.String("store", "databoxsystems/core-store", "Override core-store image")
	clearSLAdb := startCmd.Bool("flushSLAs", false, "Removes any saved apps or drivers from the SLA database so they will not restart")
	enableLogging := startCmd.Bool("v", false, "Enables verbose logging of the container-manager")
	ReGenerateDataboxCertificates := startCmd.Bool("regenerateCerts", false, "Fore databox to regenerate the databox root and certificate")

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
		log.Info("Starting Databox " + *startCmdRelease)
		opts := &databoxtype.ContainerManagerOptions{
			SwarmAdvertiseAddress:         *startCmdIP,
			ContainerManagerImage:         *cmImage,
			ArbiterImage:                  *arbiterImage,
			CoreNetworkImage:              *coreNetworkImage,
			CoreNetworkRelayImage:         *coreNetworkRelay,
			AppServerImage:                *appServerImage,
			ExportServiceImage:            *exportServerImage,
			DefaultStoreImage:             *storeImage,
			ReGenerateDataboxCertificates: *ReGenerateDataboxCertificates,
			ClearSLAs:                     *clearSLAdb,
			DefaultRegistry:               *startCmdRegistry,
			DefaultAppStore:               *appStore,
			EnableDebugLogging:            *enableLogging,
			OverridePasword:               *startCmdPassword,
		}

		databox.Start(opts)
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

		Use databox [cmd] help to see more options
		`)
}
