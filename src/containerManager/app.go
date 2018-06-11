package main

import (
	certificateGenerator "containerManager/certificateGenerator"
	containerManager "containerManager/containerManager"
	databoxStart "containerManager/databoxStart"
	log "databoxlog"
	"encoding/json"
	"io/ioutil"
	databoxTypes "lib-go-databox/types"
	"os"
)

func main() {

	DOCKER_API_VERSION := "1.35"
	os.Setenv("DOCKER_API_VERSION", DOCKER_API_VERSION)

	//get cm options from secret DATABOX_CM_OPTIONS
	cmOptionsJSON, err := ioutil.ReadFile("/run/secrets/DATABOX_CM_OPTIONS")
	log.ChkErrFatal(err)
	var options databoxTypes.ContainerManagerOptions
	err = json.Unmarshal(cmOptionsJSON, &options)
	log.ChkErrFatal(err)

	generateDataboxCertificates(options.InternalIP, options.ExternalIP)
	generateArbiterTokens()

	databox := databoxStart.New(&options)
	rootCASecretID, zmqPublic, zmqPrivate := databox.Start()
	log.Debug("key IDs :: " + rootCASecretID + " " + zmqPublic + " " + zmqPrivate)

	cm := containerManager.New(rootCASecretID, zmqPublic, zmqPrivate, &options)
	_, err = cm.WaitForContainer("arbiter", 10)
	log.ChkErrFatal(err)

	//Start the databox cm Uis and do initial configuration
	cm.Start()

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

func generateDataboxCertificates(IP string, externalIP string) {

	rootCAPath := certsBasePath + "/containerManager.crt"
	rootCAPathPub := certsBasePath + "/containerManagerPub.crt"

	if _, err := os.Stat(rootCAPath); err != nil {
		certificateGenerator.GenRootCA(rootCAPath, rootCAPathPub)
	}

	//container-manager needs extra information
	if _, err := os.Stat(certsBasePath + "/container-manager.pem"); err != nil {
		log.Debug("[generateDataboxCertificates] making cert for container-manager")
		certificateGenerator.GenCertToFile(
			rootCAPath,
			"container-manager",
			[]string{IP, externalIP, "127.0.0.1"},
			[]string{"container-manager", "localhost"},
			certsBasePath+"/container-manager.pem",
		)
	}

	components := []string{
		"databox-network",
		"export-service",
		"arbiter",
		"app-server",
	}

	for _, name := range components {
		if _, err := os.Stat(certsBasePath + "/" + name + ".pem"); err == nil {
			continue
		}
		log.Debug("[generateDataboxCertificates] making cert for " + name)
		log.Info("Making cert " + certsBasePath + "/" + name + ".pem")
		certificateGenerator.GenCertToFile(
			rootCAPath,
			name,
			[]string{"127.0.0.1"},
			[]string{name, "localhost"},
			certsBasePath+"/"+name+".pem",
		)
	}

}
