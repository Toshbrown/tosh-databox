package databoxLoader

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	log "databoxlog"
	databoxTypes "lib-go-databox/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type databoxLoader struct {
	cli     *client.Client
	version string
	path    string
	options *databoxTypes.ContainerManagerOptions
}

//TODO dose this need to be in a module or just part of the main app?
func New(version string) databoxLoader {
	cli, _ := client.NewEnvClient()

	path, _ := filepath.Abs("./")

	return databoxLoader{
		cli:     cli,
		version: version,
		path:    path,
	}
}

//func (d *databoxLoader) Start(ip, cmImage, arbiterImage, coreNetworkImage, coreNetworkRelayImage, appServerImage, exportServiceImage string, reGenerateDataboxCertificates bool, clearSLAs bool) {
func (d *databoxLoader) Start(opt *databoxTypes.ContainerManagerOptions) {

	_, err := d.cli.SwarmInit(context.Background(), swarm.InitRequest{
		ListenAddr:    "127.0.0.1",
		AdvertiseAddr: opt.SwarmAdvertiseAddress,
	})
	log.ChkErrFatal(err)

	//TODO move databox_relay creation into the CM
	os.Remove("/tmp/databox_relay")
	err = syscall.Mkfifo("/tmp/databox_relay", 0666)
	log.ChkErrFatal(err)

	d.options = opt

	d.createContainerManager()

}

func (d *databoxLoader) Stop() {

	_, err := d.cli.SwarmInspect(context.Background())
	if err != nil {
		//Not in swarm mode databox is not running
		return
	}
	filters := filters.NewArgs()
	filters.Add("label", "databox.type")

	services, err := d.cli.ServiceList(context.Background(), types.ServiceListOptions{Filters: filters})
	log.ChkErr(err)

	if len(services) > 0 {
		for _, service := range services {
			log.Info("Removing old databox service " + service.Spec.Name)
			err := d.cli.ServiceRemove(context.Background(), service.ID)
			log.ChkErr(err)
		}
	}

	d.cli.SwarmLeave(context.Background(), true)

	containers, err := d.cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filters})
	log.ChkErr(err)

	if len(containers) > 0 {
		for _, container := range containers {
			log.Info("Removing old databox container " + container.Image)
			err := d.cli.ContainerStop(context.Background(), container.ID, nil)
			log.ChkErr(err)
		}
	}

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
				Image:  d.options.ContainerManagerImage + ":" + d.version,
				Labels: map[string]string{"databox.type": "container-manager"},
				Env: []string{
					"DATABOX_ARBITER_ENDPOINT=https://arbiter:8080",
					"DATABOX_SDK=0",
					"DATABOX_VERSION=" + d.version,
					"DATABOX_DEFAULT_REGISTRY=" + d.options.DefaultRegistry,
					"DATABOX_HOST_PATH=" + d.path,
					"DATABOX_HOST_IP=" + d.options.SwarmAdvertiseAddress,
					"DATABOX_ARBITER_IMAGE=" + d.options.ArbiterImage,
					"DATABOX_CORE_NETWORK_IMAGE=" + d.options.CoreNetworkImage,
					"DATABOX_CORE_NETWORK_RELAY_IMAGE=" + d.options.CoreNetworkRelayImage,
					"DATABOX_APP_SERVER_IMAGE=" + d.options.AppServerImage,
					"DATABOX_EXPORT_SERVICE_IMAGE=" + d.options.ExportServiceImage,
					"DATABOX_REGENERATE_CERTIFICATES=" + strconv.FormatBool(d.options.ReGenerateDataboxCertificates),
					"DATABOX_FLUSH_SLA_DB=" + strconv.FormatBool(d.options.ClearSLAs),
					"DATABOX_EXTERNAL_IP=" + getExternalIP(),
					"DATABOX_ENABLE_DEBUG_LOGGING=" + strconv.FormatBool(d.options.EnableDebugLogging),
					"DATABOX_DEFAULT_STORE_IMAGE=" + d.options.DefaultStoreImage,
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

func getExternalIP() string {
	var netClient = &http.Client{
		Timeout: time.Second * 3,
	}
	response, err := netClient.Get("http://whatismyip.akamai.com/")
	log.ChkErrFatal(err)
	ip, err := ioutil.ReadAll(response.Body)
	log.ChkErrFatal(err)
	response.Body.Close()
	log.Debug("External IP found " + string(ip))
	return string(ip)
}
