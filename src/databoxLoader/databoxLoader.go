package databoxLoader

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"

	log "databoxerrors"
)

type databoxLoader struct {
	cli      *client.Client
	debug    bool
	registry string
	version  string
	path     string
}

func New(version string) databoxLoader {
	cli, _ := client.NewEnvClient()

	path, _ := filepath.Abs("./")

	return databoxLoader{
		cli:      cli,
		debug:    true,
		registry: "", //TODO fix this
		version:  version,
		path:     path,
	}
}

func (d *databoxLoader) Start(ip string) {

	_, err := d.cli.SwarmInit(context.Background(), swarm.InitRequest{
		ListenAddr:    "127.0.0.1",
		AdvertiseAddr: ip,
	})
	log.ChkErrFatal(err)

	d.createContainerManager()

}

func (d *databoxLoader) Stop() {

	filters := filters.NewArgs()
	filters.Add("label", "databox.type")

	containers, err := d.cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filters})
	log.ChkErr(err)

	if len(containers) > 0 {
		for _, container := range containers {
			fmt.Println("Removing old databox container")
			err := d.cli.ContainerStop(context.Background(), container.ID, nil)
			log.ChkErr(err)
		}
	}

	_, err = d.cli.SwarmInspect(context.Background())
	if err != nil {
		//Not in swarm mode databox is not running
		return
	}

	services, err := d.cli.ServiceList(context.Background(), types.ServiceListOptions{Filters: filters})
	log.ChkErr(err)

	if len(services) > 0 {
		for _, service := range services {
			fmt.Println("Removing old databox service")
			err := d.cli.ServiceRemove(context.Background(), service.ID)
			log.ChkErr(err)
		}
	}

	d.cli.SwarmLeave(context.Background(), true)
}

func (d *databoxLoader) createContainerManager() {

	portConfig := []swarm.PortConfig{
		swarm.PortConfig{
			TargetPort:    443,
			PublishedPort: 443,
			PublishMode:   "host",
		},
		swarm.PortConfig{
			TargetPort:    80,
			PublishedPort: 80,
			PublishMode:   "host",
		},
	}

	certsPath, _ := filepath.Abs("./certs")
	slaStorePath, _ := filepath.Abs("./slaStore")

	service := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:  d.registry + "go-container-manager:latest", // + d.version, //TODO this is hardcoded to latest for now !!
				Labels: map[string]string{"databox.type": "container-manager"},
				Env: []string{
					"DATABOX_ARBITER_ENDPOINT=https://arbiter:8080",
					"DATABOX_DEV=0", //TODO fix me
					"DATABOX_SDK=0",
					"DATABOX_VERSION=" + d.version,
					"DATABOX_HOST_PATH=" + d.path,
				},
				Mounts: []mount.Mount{
					mount.Mount{
						Type:   mount.TypeBind,
						Source: "/var/run/docker.sock",
						Target: "/var/run/docker.sock",
					},
					mount.Mount{
						Type:   mount.TypeBind,
						Source: certsPath,
						Target: "/certs",
					},
					mount.Mount{
						Type:   mount.TypeBind,
						Source: slaStorePath,
						Target: "/slaStore",
					},
				},
			},
			Placement: &swarm.Placement{
				Constraints: []string{"node.role == manager"},
			},
		},
		EndpointSpec: &swarm.EndpointSpec{
			Mode:  "dnsrr",
			Ports: portConfig,
		},
	}

	service.Name = "container-manager"

	serviceOptions := types.ServiceCreateOptions{}

	//TODO DISABLED FOR NOW
	//d.pullImage(service.TaskTemplate.ContainerSpec.Image)

	_, err := d.cli.ServiceCreate(context.Background(), service, serviceOptions)
	log.ChkErr(err)

}

func (d *databoxLoader) pullImage(image string) {

	filters := filters.NewArgs()
	filters.Add("reference", image)

	images, _ := d.cli.ImageList(context.Background(), types.ImageListOptions{Filters: filters})

	if len(images) == 0 {
		_, err := d.cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
		log.ChkErr(err)
	}
}

func (d *databoxLoader) removeContainer(name string) {
	filters := filters.NewArgs()
	filters.Add("name", name)
	containers, clerr := d.cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters,
		All:     true,
	})
	log.ChkErr(clerr)

	if len(containers) > 0 {
		rerr := d.cli.ContainerRemove(context.Background(), containers[0].ID, types.ContainerRemoveOptions{Force: true})
		log.ChkErr(rerr)
	}
}
