package coreNetworkClient

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	databoxTypes "lib-go-databox/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerNetworkTypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type CoreNetworkClient struct {
	cli     *client.Client
	request *http.Client
	CM_KEY  string
}

type NetworkConfig struct {
	NetworkName string
	DNS         string
}

func NewCoreNetworkClient(containerManagerKeyPath string, request *http.Client) CoreNetworkClient {

	cli, _ := client.NewEnvClient()

	cmKeyBytes, err := ioutil.ReadFile(containerManagerKeyPath)
	var cmKey string
	if err != nil {
		fmt.Println("Warning:: failed to read core-network CM_KEY using empty string")
		cmKey = ""
	} else {
		cmKey = b64.StdEncoding.EncodeToString([]byte(cmKeyBytes))
	}

	return CoreNetworkClient{
		cli:     cli,
		request: request,
		CM_KEY:  cmKey,
	}
}

func (cnc CoreNetworkClient) PreConfig(localContainerName string, sla databoxTypes.SLA) NetworkConfig {

	networkName := localContainerName

	internal := true
	if sla.DataboxType == "driver" {
		internal = false
	}

	//check for an existing network
	f := filters.NewArgs()
	f.Add("name", networkName)
	networkList, _ := cnc.cli.NetworkList(context.Background(), types.NetworkListOptions{Filters: f})

	var network types.NetworkResource
	var err error

	if len(networkList) > 0 {
		//network exists
		network, err = cnc.cli.NetworkInspect(context.Background(), networkList[0].ID, types.NetworkInspectOptions{})
		if err != nil {
			fmt.Println("[PreConfig] NetworkInspect1 Error ", err.Error())
		}

	} else {
		//create network
		networkCreateResponse, err := cnc.cli.NetworkCreate(context.Background(), networkName, types.NetworkCreate{
			Internal:   internal,
			Driver:     "overlay",
			Attachable: true,
		})
		if err != nil {
			fmt.Println("[PreConfig] NetworkCreate Error ", err.Error())
		}

		network, err = cnc.cli.NetworkInspect(context.Background(), networkCreateResponse.ID, types.NetworkInspectOptions{})
		if err != nil {
			fmt.Println("[PreConfig] NetworkInspect2 Error ", err.Error())
		}

		//find core network
		f := filters.NewArgs()
		f.Add("name", "databox-network") //TODO hardcoded
		coreNetwork, err := cnc.cli.ContainerList(context.Background(), types.ContainerListOptions{
			Filters: f,
		})
		if err != nil {
			fmt.Println("[PreConfig] ContainerList Error ", err.Error())
		}

		//attach to core-network
		err = cnc.cli.NetworkConnect(
			context.Background(),
			network.ID,
			coreNetwork[0].ID,
			&dockerNetworkTypes.EndpointSettings{},
		)
		if err != nil {
			fmt.Println("[PreConfig] NetworkConnect Error ", err.Error())
		}
		time.Sleep(time.Second * 5)
		//refresh network status
		network, err = cnc.cli.NetworkInspect(context.Background(), networkCreateResponse.ID, types.NetworkInspectOptions{})
		if err != nil {
			fmt.Println("[PreConfig] NetworkInspect3 Error ", err.Error())
		}
		fmt.Println("network::", network)
	}

	//find core-network IP on the new network to used as DNS
	var ipOnNewNet string
	//TODO this is wrong its not finding the IP !!!!
	for _, cont := range network.Containers {
		fmt.Println("contName=", cont.Name)
		if cont.Name == "databox-network" {
			ipOnNewNet = strings.Split(cont.IPv4Address, "/")[0]
			break
		}
	}

	fmt.Println("[PreConfig]", networkName, ipOnNewNet)

	return NetworkConfig{NetworkName: networkName, DNS: ipOnNewNet}
}

func (cnc CoreNetworkClient) post(LogFnName string, data []byte, URL string) error {
	fmt.Println("["+LogFnName+"] POSTED JSON :: ", string(data))
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("["+LogFnName+"] Error:: ", err.Error())
		return err
	}
	req.Header.Set("x-api-key", cnc.CM_KEY)
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	resp, err := cnc.request.Do(req)

	if err != nil {
		fmt.Println("["+LogFnName+"] Error ", err.Error())
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Println("["+LogFnName+"] PostError ", resp)
		return err
	}

	return nil
}
func (cnc CoreNetworkClient) ConnectEndpoints(serviceName string, peers []string) error {

	type postData struct {
		Name  string   `json:"name"`
		Peers []string `json:"peers"`
	}

	data := postData{
		Name:  serviceName,
		Peers: peers,
	}
	fmt.Println("data:: ", serviceName, peers, data)

	postBytes, _ := json.Marshal(data)
	fmt.Println("[ConnectEndpoints] POSTED JSON :: ", string(postBytes))

	return cnc.post("ConnectEndpoints", postBytes, "https://databox-network:8080/connect")
}

func (cnc CoreNetworkClient) RegisterPrivileged() error {

	cmIP, err := cnc.getCmIP()
	if err != nil {
		return err
	}

	jsonStr := "{\"src_ip\":\"" + cmIP + "\"}"
	return cnc.post("RegisterPrivileged", []byte(jsonStr), "https://databox-network:8080/privileged")

}

func (cnc CoreNetworkClient) ServiceRestart(serviceName string, oldIP string, newIP string) error {

	type postData struct {
		Name  string `json:"name"`
		OldIP string `json:"old_ip"`
		NewIP string `json:"new_ip"`
	}

	data := postData{
		Name:  serviceName,
		OldIP: oldIP,
		NewIP: newIP,
	}
	postBytes, _ := json.Marshal(data)
	return cnc.post("ServiceRestart", postBytes, "https://databox-network:8080/restart")

}

func (cnc CoreNetworkClient) getCmIP() (string, error) {

	f := filters.NewArgs()
	f.Add("name", "container-manager")

	containerList, _ := cnc.cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: f,
	})

	if len(containerList) < 1 {
		fmt.Println("[getCmIP] Error no CM found for core-network")
		return "", errors.New("No CM found for core-network")
	}

	if _, ok := containerList[0].NetworkSettings.Networks["databox-system-net"]; ok {
		return containerList[0].NetworkSettings.Networks["databox-system-net"].IPAddress, nil
	}

	fmt.Println("[getCmIP] CM not on core-network")
	return "", errors.New("CM not on core-network")

}
