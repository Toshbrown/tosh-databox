package main

import (
	certificateGenerator "containerManager/certificateGenerator"
	containerManger "containerManager/containerManager"
	databoxStart "containerManager/databoxStart"
	log "databoxlog"
	"flag"
	"fmt"
	"os"
)

func main() {

	//TODO parse CMD input for stop and dev mode etc

	DOCKER_API_VERSION := flag.String("API", "1.35", "Docker API version ")
	IP := flag.String("ip", "127.0.0.1", "The external IP to use")
	//DEV              := flag.Bool("dev", false, "Use this to enable dev mode")
	flag.Parse()

	os.Setenv("DOCKER_API_VERSION", *DOCKER_API_VERSION)

	generateDataboxCertificates(*IP)
	generateArbiterTokens()

	databox := databoxStart.New()
	rootCASecretID, zmqPublic, zmqPrivate := databox.Start()

	fmt.Println("key IDs :: ", rootCASecretID, zmqPublic, zmqPrivate)
	cm := containerManger.New(rootCASecretID, zmqPublic, zmqPrivate)

	go containerManger.ServeInsecure()
	go containerManger.ServeSecure(cm)

	fmt.Println("CM Ready and waiting")

	//Wait for a quit message
	quit := make(chan int)
	<-quit // blocks until quit is written to. Which is never for now!!
}

var certsBasePath = "./certs"

func generateArbiterTokens() {
	components := []string{
		"container-manager",
		"databox-network",
		"export-service",
		"arbiter",
	}

	if _, err := os.Stat(certsBasePath); err != nil {
		os.Mkdir(certsBasePath, 0700)
	}

	for _, name := range components {
		if _, err := os.Stat(certsBasePath + "/arbiterToken-" + name); err == nil {
			continue
		}
		certificateGenerator.GenerateArbiterTokenToFile(certsBasePath + "/arbiterToken-" + name)
	}
}

func generateDataboxCertificates(IP string) {
	rootCAPath := certsBasePath + "/containerManager.crt"

	if _, err := os.Stat(certsBasePath); err != nil {
		os.Mkdir(certsBasePath, 0700)
	}

	if _, err := os.Stat(rootCAPath); err != nil {
		certificateGenerator.GenRootCA(rootCAPath)
	}

	components := []string{
		"container-manager",
		"databox-network",
		"export-service",
		"arbiter",
		"app-server",
	}

	for _, name := range components {
		fmt.Println(name)
		if _, err := os.Stat(certsBasePath + "/" + name + ".pem"); err == nil {
			continue
		}
		log.Info("Making cert " + certsBasePath + "/" + name + ".pem")
		certificateGenerator.GenCertToFile(
			rootCAPath,
			name,
			[]string{IP, "127.0.0.1"},
			[]string{name, "localhost"},
			certsBasePath+"/"+name+".pem",
		)
	}

}
