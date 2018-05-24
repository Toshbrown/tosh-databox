package containerManager

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"certificateGenerator"
	"lib-go-databox/arbiterClient"
	"lib-go-databox/coreNetworkClient"
	"lib-go-databox/databoxRequest"
	databoxTypes "lib-go-databox/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"

	log "databoxerrors"
)

type ContainerManager struct {
	cli                *client.Client
	ArbiterClient      arbiterClient.ArbiterClient
	CoreNetworkClient  coreNetworkClient.CoreNetworkClient
	Request            *http.Client
	DATABOX_ROOT_CA_ID string
	ZMQ_PUBLIC_KEY_ID  string
	ZMQ_PRIVATE_KEY_ID string
	ARCH               string
	Version            string
}

func NewContainerManager(rootCASecretId string, zmqPublicId string, zmqPrivateId string) ContainerManager {

	cli, _ := client.NewEnvClient()

	request := databoxRequest.NewDataboxHTTPsAPIWithPaths("/certs/containerManager.crt")
	ac := arbiterClient.NewArbiterClient("/certs/arbiterToken-container-manager", request, "https://arbiter:8080")
	cnc := coreNetworkClient.NewCoreNetworkClient("/certs/arbiterToken-databox-network", request)

	cm := ContainerManager{
		cli:                cli,
		ArbiterClient:      ac,
		CoreNetworkClient:  cnc,
		Request:            request,
		DATABOX_ROOT_CA_ID: rootCASecretId,
		ZMQ_PUBLIC_KEY_ID:  zmqPublicId,
		ZMQ_PRIVATE_KEY_ID: zmqPrivateId,
		Version:            os.Getenv("DATABOX_VERSION"),
	}

	if os.Getenv("GOARCH") == "arm" {
		cm.ARCH = "arm"
	} else {
		cm.ARCH = ""
	}

	//register with core-network
	cnc.RegisterPrivileged()

	return cm
}

func (cm ContainerManager) LaunchFromSLA(sla databoxTypes.SLA) error {

	//start the container
	switch sla.DataboxType {
	case databoxTypes.DataboxTypeApp:
		cm.launchApp(sla)
	case databoxTypes.DataboxTypeDriver:
		cm.launchDriver(sla)
	}

	return nil
}

func (cm ContainerManager) launchDriver(sla databoxTypes.SLA) {

	registry := "databoxsystems/" //TODO this needs to be set from the SLA?

	localContainerName := sla.Name + cm.ARCH

	//set up the networks.
	netConf := cm.CoreNetworkClient.PreConfig(localContainerName, sla)

	service := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Labels: map[string]string{"databox.type": "driver"},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:  registry + sla.Name + ":" + cm.Version,
				Labels: map[string]string{"databox.type": "driver"},

				Env: []string{
					"DATABOX_ARBITER_ENDPOINT=https://arbiter:8080",
					"DATABOX_LOCAL_NAME=" + localContainerName,
				},
				Secrets: cm.addSecrets(localContainerName, sla.DataboxType),
				DNSConfig: &swarm.DNSConfig{
					Nameservers: []string{netConf.DNS},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{
				Target: netConf.NetworkName,
			}},
		},
	}

	service.Name = localContainerName

	serviceOptions := types.ServiceCreateOptions{}

	storeName := cm.launchStore(sla, netConf)

	service.TaskTemplate.ContainerSpec.Env = append(service.TaskTemplate.ContainerSpec.Env,
		"DATABOX_ZMQ_ENDPOINT=tcp://"+storeName+":5555")
	service.TaskTemplate.ContainerSpec.Env = append(service.TaskTemplate.ContainerSpec.Env,
		"DATABOX_ZMQ_DEALER_ENDPOINT=tcp://"+storeName+":5556")

	_, err := cm.cli.ServiceCreate(context.Background(), service, serviceOptions)
	if err != nil {
		log.Err("[launchDriver] Error launching " + localContainerName + " " + err.Error())
	}

	cm.CoreNetworkClient.ConnectEndpoints(localContainerName, []string{storeName})

	cm.addPermissionsFromSla(sla)
}

func (cm ContainerManager) launchApp(sla databoxTypes.SLA) {

	registry := "databoxsystems/" //TODO this needs to be set from the SLA?

	localContainerName := sla.Name + cm.ARCH

	//set up the networks.
	netConf := cm.CoreNetworkClient.PreConfig(localContainerName, sla)

	service := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Labels: map[string]string{"databox.type": "app"},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:  registry + sla.Name + ":" + cm.Version,
				Labels: map[string]string{"databox.type": "app"},

				Env: []string{
					"DATABOX_ARBITER_ENDPOINT=https://arbiter:8080",
					"DATABOX_LOCAL_NAME=" + localContainerName,
					"DATABOX_EXPORT_SERVICE_ENDPOINT=https://export-service:8080",
				},
				Secrets: cm.addSecrets(localContainerName, sla.DataboxType),
				DNSConfig: &swarm.DNSConfig{
					Nameservers: []string{netConf.DNS},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{
				Target: netConf.NetworkName,
			}},
		},
	}

	service.Name = localContainerName

	serviceOptions := types.ServiceCreateOptions{}

	//add datasource info to the env vars and crate a list of stores to assess
	requiredStores := map[string]string{}
	for _, ds := range sla.Datasources {
		hypercatString, _ := json.Marshal(ds.Hypercat)
		service.TaskTemplate.ContainerSpec.Env = append(
			service.TaskTemplate.ContainerSpec.Env,
			"DATASOURCE_"+ds.Clientid+"="+string(hypercatString),
		)
		parsedURL, _ := url.Parse(ds.Hypercat.Href)
		storeName := parsedURL.Hostname()
		requiredStores[storeName] = storeName
	}

	//launch a store if required
	if sla.ResourceRequirements.Store != "" {
		storeName := cm.launchStore(sla, netConf)

		service.TaskTemplate.ContainerSpec.Env = append(service.TaskTemplate.ContainerSpec.Env,
			"DATABOX_ZMQ_ENDPOINT=tcp://"+storeName+":5555")
		service.TaskTemplate.ContainerSpec.Env = append(service.TaskTemplate.ContainerSpec.Env,
			"DATABOX_ZMQ_DEALER_ENDPOINT=tcp://"+storeName+":5556")

		requiredStores[storeName] = storeName

	}

	_, err := cm.cli.ServiceCreate(context.Background(), service, serviceOptions)
	if err != nil {
		log.Err("[launchApp] Error launching " + localContainerName + " " + err.Error())
	}

	//connect to stores
	if len(requiredStores) > 0 {
		storesToConnect := []string{}
		for store := range requiredStores {
			storesToConnect = append(storesToConnect, store)
		}
		fmt.Println("storesToConnect", storesToConnect)
		cm.CoreNetworkClient.ConnectEndpoints(localContainerName, storesToConnect)
	}

	cm.addPermissionsFromSla(sla)

}

func (cm ContainerManager) launchStore(sla databoxTypes.SLA, netConf coreNetworkClient.NetworkConfig) string {

	registry := "databoxsystems/" //TODO this needs to be set from the SLA?

	requiredStore := sla.ResourceRequirements.Store

	//TODO check store is supported!!

	requiredStoreName := sla.Name + "-" + requiredStore + cm.ARCH

	service := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Labels: map[string]string{"databox.type": "store"},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:  registry + requiredStore + ":" + cm.Version,
				Labels: map[string]string{"databox.type": "store"},
				Env: []string{
					"DATABOX_ARBITER_ENDPOINT=https://arbiter:8080",
					"DATABOX_LOCAL_NAME=" + requiredStoreName,
				},
				Secrets: cm.addSecrets(requiredStoreName, databoxTypes.DataboxType("store")),
				DNSConfig: &swarm.DNSConfig{
					Nameservers: []string{netConf.DNS},
				},
				Mounts: []mount.Mount{
					mount.Mount{
						Source: requiredStoreName,
						Target: "/database",
						Type:   "volume",
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{
				Target: netConf.NetworkName,
			}},
		},
	}

	service.Name = requiredStoreName
	serviceOptions := types.ServiceCreateOptions{}

	storeName := requiredStoreName

	_, err := cm.cli.ServiceCreate(context.Background(), service, serviceOptions)
	if err != nil {
		log.Err("Launching store " + requiredStoreName + " " + err.Error())
	}

	return storeName
}

func (cm ContainerManager) addSecrets(containerName string, databoxType databoxTypes.DataboxType) []*swarm.SecretReference {

	createSecret := func(name string, data []byte, filename string) *swarm.SecretReference {
		secret := swarm.SecretSpec{
			Annotations: swarm.Annotations{
				Name: name,
			},
			Data: data,
		}
		secretCreateResponse, err := cm.cli.SecretCreate(context.Background(), secret)
		if err != nil {
			log.Err("addSecrets createSecret " + err.Error())
		}
		return &swarm.SecretReference{
			SecretID:   secretCreateResponse.ID,
			SecretName: name,
			File: &swarm.SecretReferenceFileTarget{
				Name: filename,
				UID:  "0",
				GID:  "0",
				Mode: 0444,
			},
		}
	}

	secrets := []*swarm.SecretReference{}

	secrets = append(secrets, &swarm.SecretReference{
		SecretID:   cm.DATABOX_ROOT_CA_ID,
		SecretName: "DATABOX_ROOT_CA",
		File: &swarm.SecretReferenceFileTarget{
			Name: "DATABOX_ROOT_CA",
			UID:  "0",
			GID:  "0",
			Mode: 0444,
		},
	})

	secrets = append(secrets, &swarm.SecretReference{
		SecretID:   cm.ZMQ_PUBLIC_KEY_ID,
		SecretName: "ZMQ_PUBLIC_KEY",
		File: &swarm.SecretReferenceFileTarget{
			Name: "ZMQ_PUBLIC_KEY",
			UID:  "0",
			GID:  "0",
			Mode: 0444,
		},
	})

	cert := certificateGenerator.GenCert(
		"./certs/containerManager.crt", //TODO Fix this
		containerName,
		[]string{"127.0.0.1"},
		[]string{containerName},
	)
	secrets = append(secrets, createSecret(strings.ToUpper(containerName)+".pem", cert, "DATABOX.pem"))

	rawToken := certificateGenerator.GenerateArbiterToken()
	b64TokenString := b64.StdEncoding.EncodeToString(rawToken)
	secrets = append(secrets, createSecret(strings.ToUpper(containerName)+"_KEY", rawToken, "ARBITER_TOKEN"))

	//update the arbiter with the containers token
	err := cm.ArbiterClient.UpdateArbiter(containerName, b64TokenString, databoxType)
	if err != nil {
		log.Err("Add Secrets error updating arbiter " + err.Error())
	}

	//Only pass the zmq private key to stores.
	if databoxType == "store" {
		log.Info("[addSecrets] ZMQ_PRIVATE_KEY_ID=" + cm.ZMQ_PRIVATE_KEY_ID)
		secrets = append(secrets, &swarm.SecretReference{
			SecretID:   cm.ZMQ_PRIVATE_KEY_ID,
			SecretName: "ZMQ_SECRET_KEY",
			File: &swarm.SecretReferenceFileTarget{
				Name: "ZMQ_SECRET_KEY",
				UID:  "0",
				GID:  "0",
				Mode: 0444,
			},
		})
	}

	return secrets
}

func (cm ContainerManager) addPermissionsFromSla(sla databoxTypes.SLA) {

	var err error

	localContainerName := sla.Name + cm.ARCH

	//set export permissions from export-whitelist
	if len(sla.ExportWhitelists) > 0 {

		urlsString := "destination = \""
		for i, whiteList := range sla.ExportWhitelists {
			urlsString = urlsString + whiteList.Url
			if i < len(sla.ExportWhitelists) {
				urlsString = urlsString + ","
			}
		}
		urlsString = urlsString + "\""

		log.Info("Adding Export permissions for " + localContainerName + " on " + urlsString)

		err = cm.addPermission(localContainerName, "export-service", "/export/", "POST", []string{urlsString})
		if err != nil {
			log.Err("Adding write permissions for Actuator " + err.Error())
		}
	}

	//set read permissions from the sla for DATASOURCES.
	if sla.DataboxType == "app" && len(sla.Datasources) > 0 {

		for _, ds := range sla.Datasources {
			datasourceEndpoint, _ := url.Parse(ds.Hypercat.Href)
			datasourceName := datasourceEndpoint.Path

			//Deal with Actuators
			isActuator := false
			for _, item := range ds.Hypercat.ItemMetadata {
				switch item.(type) {
				case databoxTypes.RelValPairBool:
					if item.(databoxTypes.RelValPairBool).Rel == "urn:X-databox:rels:isActuator" && item.(databoxTypes.RelValPairBool).Val == true {
						isActuator = true
						break
					}
				default:
					// we are only interested in databoxTypes.RelValPairBool
				}
			}
			if isActuator == true {
				log.Info("Adding write permissions for Actuator " + datasourceName + " on " + datasourceEndpoint.Hostname())
				err = cm.addPermission(localContainerName, datasourceEndpoint.Hostname(), "/"+datasourceName+"/*", "POST", []string{})
				if err != nil {
					log.Err("Adding write permissions for Actuator " + err.Error())
				}
			}

			log.Info("Adding read permissions for /status  on " + datasourceEndpoint.Hostname())
			err = cm.addPermission(localContainerName, datasourceEndpoint.Hostname(), "/status", "GET", []string{})
			if err != nil {
				log.Err("Adding write permissions for Datasource " + err.Error())
			}

			log.Info("Adding read permissions for " + localContainerName + " on data source " + datasourceName + " on " + datasourceEndpoint.Hostname())
			err = cm.addPermission(localContainerName, datasourceEndpoint.Hostname(), datasourceName, "GET", []string{})
			if err != nil {
				log.Err("Adding write permissions for Datasource " + err.Error())
			}

			log.Info("Adding read permissions for " + localContainerName + " on data source " + datasourceName + " on " + datasourceEndpoint.Hostname() + "/*")
			err = cm.addPermission(localContainerName, datasourceEndpoint.Hostname(), datasourceName+"/*", "GET", []string{})
			if err != nil {
				log.Err("Adding write permissions for Datasource " + err.Error())
			}

		}

	}

	//Add permissions for dependent stores if needed for apps and drivers
	if sla.ResourceRequirements.Store != "" {
		requiredStoreName := sla.Name + "-" + sla.ResourceRequirements.Store + cm.ARCH

		log.Info("Adding read permissions for container-manager on " + requiredStoreName + "/cat")
		err = cm.addPermission("container-manager", requiredStoreName, "/cat", "GET", []string{})
		if err != nil {
			log.Err("Adding write permissions for Actuator " + err.Error())
		}

		log.Info("Adding write permissions for dependent store " + localContainerName + " on " + requiredStoreName + "/*")
		err = cm.addPermission(localContainerName, requiredStoreName, "/*", "POST", []string{})
		if err != nil {
			log.Err("Adding write permissions for Actuator " + err.Error())
		}

		err = cm.addPermission(localContainerName, requiredStoreName, "/*", "GET", []string{})
		if err != nil {
			log.Err("Adding write permissions for Actuator " + err.Error())
		}

	}
}

func (cm ContainerManager) addPermission(name string, target string, path string, method string, caveats []string) error {

	newPermission := arbiterClient.ContainerPermissions{
		Name: name,
		Route: arbiterClient.Route{
			Target: target,
			Path:   path,
			Method: method,
		},
		Caveats: caveats,
	}

	return cm.ArbiterClient.GrantContainerPermissions(newPermission)

}

func (cm ContainerManager) Restart(name string) error {
	filters := filters.NewArgs()
	filters.Add("label", "com.docker.swarm.service.name="+name)

	contList, _ := cm.cli.ContainerList(context.Background(),
		types.ContainerListOptions{
			Filters: filters,
		})
	if len(contList) < 1 {
		return errors.New("Service " + name + " not running")
	}

	//Stash the old container IP
	oldIP := ""
	for netName := range contList[0].NetworkSettings.Networks {
		serviceName := strings.Replace(name, "-core-store", "", 1)
		if strings.Contains(netName, serviceName) {
			oldIP = contList[0].NetworkSettings.Networks[netName].IPAMConfig.IPv4Address
		}
	}

	//Stop the container then the service will start a new one
	err := cm.cli.ContainerRemove(context.Background(), contList[0].ID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return errors.New("Cannot remove " + name + " " + err.Error())
	}

	//wait for the restarted container to start
	var newContList []types.Container
	loopCount := 0
	for {

		newContList, _ = cm.cli.ContainerList(context.Background(),
			types.ContainerListOptions{
				Filters: filters,
			})

		if len(newContList) < 1 {
			time.Sleep(time.Second)
			loopCount++
			if loopCount > 10 {
				return errors.New("Service has not restarted after 10 seconds !! Could not update corenetwork IP")
			}
			continue
		}

		break
	}

	//found restarted container !!!
	//Stash the new container IP
	newIP := ""
	for netName := range newContList[0].NetworkSettings.Networks {
		serviceName := strings.Replace(name, "-core-store", "", 1)
		if strings.Contains(netName, serviceName) {
			newIP = newContList[0].NetworkSettings.Networks[netName].IPAMConfig.IPv4Address
			log.Debug("IP found " + newIP)
		}
	}

	return cm.CoreNetworkClient.ServiceRestart(name, oldIP, newIP)
}

func (cm ContainerManager) Uninstall(name string) error {
	serFilters := filters.NewArgs()
	serFilters.Add("name", name)

	serList, _ := cm.cli.ServiceList(context.Background(),
		types.ServiceListOptions{
			Filters: serFilters,
		})
	if len(serList) < 1 {
		return errors.New("Service " + name + " not running")
	}

	err := cm.cli.ServiceRemove(context.Background(), serList[0].ID)

	//remove secrets
	secFilters := filters.NewArgs()
	secFilters.Add("name", strings.ToUpper(name)+".pem")
	secFilters.Add("name", strings.ToUpper(name)+"_KEY")

	secretList, err := cm.cli.SecretList(context.Background(), types.SecretListOptions{
		Filters: secFilters,
	})
	for _, s := range secretList {
		cm.cli.SecretRemove(context.Background(), s.ID)
	}

	return err
}
