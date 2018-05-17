package databoxLogParser

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	derr "databoxerrors"
)

func ShowLogs() {

	cli, _ := client.NewEnvClient()

	fmt.Println("outputting logs")

	//filters := filters.NewArgs()
	//filters.Add("label", "databox.type")
	//services, err := d.cli.ServiceList(context.Background(), types.ServiceListOptions{Filters: filters})
	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
	derr.ChkErr(err)

	logChan := make(chan string)
	var wg sync.WaitGroup

	for _, service := range services {
		fmt.Println("Monitoring logs for ", service.Spec.Name)
		ioLogReader, err := cli.ServiceLogs(context.Background(), service.ID,
			types.ContainerLogsOptions{
				Follow:     true,
				ShowStderr: true,
				ShowStdout: true,
				Tail:       "all",
				Timestamps: true,
			},
		)

		if err != nil {
			fmt.Println("Error reading logs for ", service.Spec.Name)
			continue
		}
		wg.Add(1)
		go func(name string) {
			for {
				message := make([]byte, 128)
				n, err := ioLogReader.Read(message)
				if err != nil {
					wg.Done()
					ioLogReader.Close()
					break
				}
				if n > 0 {
					logChan <- name + "\t: " + string(message)
				}
			}
		}(service.Spec.Name)
	}

	go func() {
		for {
			select {
			case msg := <-logChan:
				fmt.Print(msg)
			}
		}
	}()

	wg.Wait()
}
