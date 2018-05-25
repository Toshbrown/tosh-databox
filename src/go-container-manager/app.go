package main

import (
	certificateGenerator "certificateGenerator"
	containerManager "containerManager"
	log "databoxerrors"
	"flag"
	"fmt"
	"os"
)

func main() {

	//TODO parse CMD input for stop and dev mode etc

	DOCKER_API_VERSION := flag.String("API", "1.35", "Docker API version ")
	IP := flag.String("ip", "192.168.1.131", "The external IP to use")
	//DEV              := flag.Bool("dev", false, "Use this to enable dev mode")
	flag.Parse()

	os.Setenv("DOCKER_API_VERSION", *DOCKER_API_VERSION)

	generateDataboxCertificates(*IP)
	generateArbiterTokens()

	databox := containerManager.NewDatabox()
	rootCASecretID, zmqPublic, zmqPrivate := databox.Start()

	fmt.Println("key IDs :: ", rootCASecretID, zmqPublic, zmqPrivate)
	cm := containerManager.NewContainerManager(rootCASecretID, zmqPublic, zmqPrivate)

	go containerManager.ServeInsecure()
	go containerManager.ServeSecure(cm)

	fmt.Println("CM Ready and watling")

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
			//fmt.Println("Cert exists, delete the  " + certsBasePath + "/" + name + ".pem to regenerate")
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
