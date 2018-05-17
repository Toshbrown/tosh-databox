package containerManager

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	zmq "github.com/pebbe/zmq4"
)

type Databox struct {
	cli                           *client.Client
	cliErr                        []error
	debug                         bool
	registry                      string
	version                       string
	hostPath                      string
	DATABOX_ROOT_CA_ID            string
	CM_KEY_ID                     string
	DATABOX_ARBITER_ID            string
	DATABOX_EXPORT_SERVICE_KEY_ID string
	ZMQ_SECRET_KEY_ID             string
	ZMQ_PUBLIC_KEY_ID             string
	DATABOX_DNS_IP                string
	DATABOX_PEM                   string
	DATABOX_NETWORK_KEY           string
	DontPull                      bool
}

func NewDatabox() Databox {
	cli, _ := client.NewEnvClient()

	return Databox{
		cli:      cli,
		cliErr:   nil,
		debug:    true,
		registry: "databoxsystems",
		version:  os.Getenv("DATABOX_VERSION"),
		hostPath: os.Getenv("DATABOX_HOST_PATH"),
		DontPull: true, //dont pull images for now
	}
}

func (d *Databox) setErr(err error) {
	if err == nil {
		return
	}
	if d.debug == true {
		fmt.Println("[Databox Error] ", err)
	}
	d.cliErr = append(d.cliErr, err)
}

func (d *Databox) Start() (string, string, string) {

	fmt.Println("CM STARTED")
	//start the core containers
	d.startCoreNetwork()

	//SET CM DNS, create secrets and join to databox-system-net
	d.updateContainerManager()

	//Create global secrets that are used in more than one container
	fmt.Println("Creating secrets")
	d.DATABOX_ROOT_CA_ID = d.createSecretFromFile("DATABOX_ROOT_CA", "./certs/containerManager.crt")
	d.CM_KEY_ID = d.createSecretFromFile("CM_KEY", "./certs/arbiterToken-container-manager")
	d.DATABOX_ARBITER_ID = d.createSecretFromFile("DATABOX_ARBITER.pem", "./certs/arbiter.pem")
	d.DATABOX_EXPORT_SERVICE_KEY_ID = d.createSecretFromFile("DATABOX_EXPORT_SERVICE_KEY", "./certs/arbiterToken-export-service")

	d.DATABOX_PEM = d.createSecretFromFile("DATABOX.pem", "./certs/container-manager.pem") //TODO sort out certs!!
	d.DATABOX_NETWORK_KEY = d.createSecretFromFile("DATABOX_NETWORK_KEY", "./certs/arbiterToken-databox-network")

	//get the ZMQ secret IDs to pass to other containers
	filters := filters.NewArgs()
	filters.Add("name", "ZMQ_")
	zmqsecrests, _ := d.cli.SecretList(context.Background(), types.SecretListOptions{Filters: filters})
	for _, secret := range zmqsecrests {
		if secret.Spec.Name == "ZMQ_PUBLIC_KEY" {
			d.ZMQ_PUBLIC_KEY_ID = secret.ID
		}
		if secret.Spec.Name == "ZMQ_SECRET_KEY" {
			d.ZMQ_SECRET_KEY_ID = secret.ID
		}
	}

	d.startAppServer()
	d.startArbiter()
	d.startExportService()

	return d.DATABOX_ROOT_CA_ID, d.ZMQ_PUBLIC_KEY_ID, d.ZMQ_SECRET_KEY_ID
}

func (d *Databox) getDNSIP() (string, error) {

	filters := filters.NewArgs()
	filters.Add("name", "databox-network")
	contList, _ := d.cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters,
	})

	//after CM update we do not need to do this again!!
	if len(contList) > 0 {
		//store the databox-network IP to pass as dns server
		containerJSON, _ := d.cli.ContainerInspect(context.Background(), contList[0].ID)
		fmt.Println("getDNSIP found ip: ", containerJSON.NetworkSettings.Networks["databox-system-net"].IPAddress)
		return containerJSON.NetworkSettings.Networks["databox-system-net"].IPAddress, nil
	}

	fmt.Println("getDNSIP ip no found")
	return "", errors.New("databox-network not found")
}

func (d *Databox) startCoreNetwork() {

	if d.cliErr != nil {
		return
	}

	filters := filters.NewArgs()
	filters.Add("name", "databox-network")

	contList, _ := d.cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters,
	})

	//after CM update we do not need to do this again!!
	if len(contList) > 0 {
		fmt.Println("databox-network already running")
		//store the databox-network IP to pass as dns server
		d.DATABOX_DNS_IP, _ = d.getDNSIP()
		return
	}

	fmt.Println("STARTING databox-network")

	options := types.NetworkCreate{
		Driver:     "overlay",
		Attachable: true,
		Internal:   false,
	}

	_, err := d.cli.NetworkCreate(context.Background(), "databox-system-net", options)
	d.setErr(err)

	config := &container.Config{
		Image:  d.registry + "/core-network:0.3.2", // + d.version, //TODO fix this core network was split in > 0.3.2
		Labels: map[string]string{"databox.type": "databox-network"},
	}

	tokenPath := d.hostPath + "/certs/arbiterToken-databox-network"
	pemPath := d.hostPath + "/certs/databox-network.pem"

	hostConfig := &container.HostConfig{
		Binds: []string{
			tokenPath + ":/run/secrets/DATABOX_NETWORK_KEY:rw",
			pemPath + ":/run/secrets/DATABOX_NETWORK.pem:rw",
		},
		CapAdd: []string{"NET_ADMIN"},
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"databox-system-net": &network.EndpointSettings{
				Aliases: []string{"databox-network"},
			},
		},
	}
	containerName := "databox-network"

	d.removeContainer(containerName)

	d.pullImage(config.Image)

	containerCreateCreatedBody, ccErr := d.cli.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, containerName)
	d.setErr(ccErr)

	d.cli.ContainerStart(context.Background(), containerCreateCreatedBody.ID, types.ContainerStartOptions{})
	d.DATABOX_DNS_IP, _ = d.getDNSIP()
}

func (d *Databox) pullImage(image string) {

	filters := filters.NewArgs()
	filters.Add("reference", image)

	images, _ := d.cli.ImageList(context.Background(), types.ImageListOptions{Filters: filters})

	if len(images) == 0 {
		_, err := d.cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
		d.setErr(err)
	}
}

func (d *Databox) updateContainerManager() {

	//TODO error checking ;-)

	d.DATABOX_DNS_IP, _ = d.getDNSIP()

	filters := filters.NewArgs()
	filters.Add("name", "container-manager")

	swarmService, _ := d.cli.ServiceList(context.Background(), types.ServiceListOptions{
		Filters: filters,
	})

	if swarmService[0].Spec.TaskTemplate.ContainerSpec.DNSConfig != nil {
		//we have already updated the service!!!
		fmt.Println("container-manager service is up to date")

		f, _ := os.OpenFile("/ect/resolv.conf", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		defer f.Close()
		f.WriteString("nameserver " + d.DATABOX_DNS_IP)

		return
	}

	//make ZMQ secrests
	public, private, zmqErr := zmq.NewCurveKeypair()
	d.setErr(zmqErr)
	d.ZMQ_PUBLIC_KEY_ID = d.createSecret("ZMQ_PUBLIC_KEY", public)
	d.ZMQ_SECRET_KEY_ID = d.createSecret("ZMQ_SECRET_KEY", private)

	fmt.Println("Updating container-manager Service", d.DATABOX_DNS_IP)

	swarmService[0].Spec.TaskTemplate.ContainerSpec.DNSConfig = &swarm.DNSConfig{
		Nameservers: []string{d.DATABOX_DNS_IP},
		Options:     []string{"ndots:0"},
	}

	swarmService[0].Spec.TaskTemplate.Networks = []swarm.NetworkAttachmentConfig{
		swarm.NetworkAttachmentConfig{
			Target: "databox-system-net",
		},
	}
	swarmService[0].Spec.TaskTemplate.ContainerSpec.Secrets = append(
		swarmService[0].Spec.TaskTemplate.ContainerSpec.Secrets,
		&swarm.SecretReference{
			SecretID:   d.ZMQ_PUBLIC_KEY_ID,
			SecretName: "ZMQ_PUBLIC_KEY",
			File: &swarm.SecretReferenceFileTarget{
				Name: "ZMQ_PUBLIC_KEY",
				UID:  "0",
				GID:  "0",
				Mode: 929,
			},
		})

	_, err := d.cli.ServiceUpdate(
		context.Background(),
		swarmService[0].ID,
		swarmService[0].Version,
		swarmService[0].Spec,
		types.ServiceUpdateOptions{},
	)
	d.setErr(err)

	//waiting to be rebooted
	time.Sleep(time.Second * 100)

}

func (d *Databox) startAppServer() {

	containerName := "app-server"

	config := &container.Config{
		Image: d.registry + "/" + containerName + ":latest",
		Env:   []string{"LOCAL_MODE=1", "PORT=8181"},
		ExposedPorts: nat.PortSet{
			"8181/tcp": {},
		},
		Labels: map[string]string{"databox.type": "app-server"},
	}

	pemPath := d.hostPath + "/certs/app-server.pem"

	ports := make(nat.PortMap)
	ports["8181/tcp"] = []nat.PortBinding{nat.PortBinding{HostPort: "8181"}}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			mount.Mount{
				Type:   mount.TypeBind,
				Source: pemPath,
				Target: "/run/secrets/DATABOX.pem",
			},
		},
		PortBindings: ports,
	}
	networkingConfig := &network.NetworkingConfig{}

	d.removeContainer(containerName)

	d.pullImage(config.Image)

	containerCreateCreatedBody, ccErr := d.cli.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, containerName)
	d.setErr(ccErr)

	d.cli.ContainerStart(context.Background(), containerCreateCreatedBody.ID, types.ContainerStartOptions{})
}

func (d *Databox) startExportService() {
	s1ID := d.createSecretFromFile("DATABOX_EXPORT_SERVICE.pem", "./certs/export-service.pem")

	service := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: d.registry + "/export-service:" + d.version,
				Env:   []string{"DATABOX_ARBITER_ENDPOINT=https://arbiter:8080"},
				Secrets: []*swarm.SecretReference{
					&swarm.SecretReference{
						SecretID:   d.DATABOX_ROOT_CA_ID,
						SecretName: "DATABOX_ROOT_CA",
						File: &swarm.SecretReferenceFileTarget{
							Name: "DATABOX_ROOT_CA",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
					&swarm.SecretReference{
						SecretID:   s1ID,
						SecretName: "DATABOX_EXPORT_SERVICE.pem",
						File: &swarm.SecretReferenceFileTarget{
							Name: "DATABOX_EXPORT_SERVICE.pem",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
					&swarm.SecretReference{
						SecretID:   d.DATABOX_EXPORT_SERVICE_KEY_ID,
						SecretName: "DATABOX_EXPORT_SERVICE_KEY",
						File: &swarm.SecretReferenceFileTarget{
							Name: "DATABOX_EXPORT_SERVICE_KEY",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{
				Target: "databox-system-net",
			}},
		},
	}

	service.Name = "export-service"

	serviceOptions := types.ServiceCreateOptions{}

	d.pullImage(service.TaskTemplate.ContainerSpec.Image)

	_, err := d.cli.ServiceCreate(context.Background(), service, serviceOptions)
	d.setErr(err)

}

func (d *Databox) startArbiter() {

	service := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: d.registry + "/arbiter:" + d.version,
				Secrets: []*swarm.SecretReference{
					&swarm.SecretReference{
						SecretID:   d.DATABOX_ROOT_CA_ID,
						SecretName: "DATABOX_ROOT_CA",
						File: &swarm.SecretReferenceFileTarget{
							Name: "DATABOX_ROOT_CA",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
					&swarm.SecretReference{
						SecretID:   d.CM_KEY_ID,
						SecretName: "CM_KEY",
						File: &swarm.SecretReferenceFileTarget{
							Name: "CM_KEY",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
					&swarm.SecretReference{
						SecretID:   d.DATABOX_ARBITER_ID,
						SecretName: "DATABOX_ARBITER.pem",
						File: &swarm.SecretReferenceFileTarget{
							Name: "DATABOX_ARBITER.pem",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
					&swarm.SecretReference{
						SecretID:   d.DATABOX_EXPORT_SERVICE_KEY_ID,
						SecretName: "DATABOX_EXPORT_SERVICE_KEY",
						File: &swarm.SecretReferenceFileTarget{
							Name: "DATABOX_EXPORT_SERVICE_KEY",
							UID:  "0",
							GID:  "0",
							Mode: 929,
						},
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{
				Target: "databox-system-net",
			}},
		},
	}

	service.Name = "arbiter"

	serviceOptions := types.ServiceCreateOptions{}

	d.pullImage(service.TaskTemplate.ContainerSpec.Image)

	_, err := d.cli.ServiceCreate(context.Background(), service, serviceOptions)
	d.setErr(err)

}

func (d *Databox) createSecret(name, data string) string {

	secret := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		Data: []byte(data),
	}
	secretCreateResponse, err := d.cli.SecretCreate(context.Background(), secret)
	d.setErr(err)

	return secretCreateResponse.ID
}

func (d *Databox) createSecretFromFile(name, dataPath string) string {

	data, _ := ioutil.ReadFile(dataPath)
	secret := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		Data: data,
	}
	secretCreateResponse, err := d.cli.SecretCreate(context.Background(), secret)
	d.setErr(err)

	return secretCreateResponse.ID
}

func (d *Databox) removeContainer(name string) {
	filters := filters.NewArgs()
	filters.Add("name", name)
	containers, clerr := d.cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters,
		All:     true,
	})
	d.setErr(clerr)

	if len(containers) > 0 {
		rerr := d.cli.ContainerRemove(context.Background(), containers[0].ID, types.ContainerRemoveOptions{Force: true})
		d.setErr(rerr)
	}
}
