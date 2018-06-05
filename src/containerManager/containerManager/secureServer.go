package containerManager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"lib-go-databox/coreStoreClient"
	databoxTypes "lib-go-databox/types"

	"containerManager/databoxAuthMiddleware"

	log "databoxlog"

	"containerManager/databoxProxyMiddleware"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gorilla/mux"
)

var dboxproxy *databoxProxyMiddleware.DataboxProxyMiddleware

func ServeSecure(cm ContainerManager) {

	//pull required databox components from the ContainerManager
	cli := cm.cli
	ac := cm.ArbiterClient
	request := cm.Request

	//start the https server for the app UI
	r := mux.NewRouter()

	dboxproxy = databoxProxyMiddleware.New("/certs/containerManager.crt")
	//proxy to the arbiter ui
	dboxproxy.Add("arbiter")

	dboxauth := databoxAuthMiddleware.New("qwertyuiop", dboxproxy)

	log.Debug("Installing databox Middlewares")
	r.Use(dboxauth.AuthMiddleware, dboxproxy.ProxyMiddleware)

	r.HandleFunc("/api/cmlogs", func(w http.ResponseWriter, r *http.Request) {
		jsonString := cm.Logger.GetLastNLogEntriesRaw(100)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonString)
	}).Methods("GET")

	r.HandleFunc("/api/datasource/list", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("/api/datasource/list called")
		hyperCatRoot, err := ac.GetRootDataSourceCatalogue()
		if err != nil {
			log.Err("/api/datasource/list GetRootDataSourceCatalogue " + err.Error())
		}

		//hcr, _ := json.Marshal(hyperCatRoot)
		//log.Debug("/api/datasource/list hyperCatRoot=" + string(hcr))
		var datasources []databoxTypes.HypercatItem
		for _, item := range hyperCatRoot.Items {
			//get the store cat
			storeURL, _ := coreStoreClient.GetStoreURLFromDsHref(item.Href)
			sc := coreStoreClient.New(request, &ac, "/run/secrets/ZMQ_PUBLIC_KEY", storeURL, false)
			storeCat, err := sc.GetStoreDataSourceCatalogue(item.Href)
			if err != nil {
				log.Err("[/api/datasource/list] Error GetStoreDataSourceCatalogue " + err.Error())
			}
			//src, _ := json.Marshal(storeCat)
			//log.Debug("/api/datasource/list got store cat: " + string(src))
			//build the datasource list
			for _, ds := range storeCat.Items {
				log.Debug("/api/datasource/list " + ds.Href)
				datasources = append(datasources, ds)
			}
		}
		jsonString, err := json.Marshal(datasources)
		if err != nil {
			log.Err("[/api/datasource/list] Error " + err.Error())
		}
		log.Debug("[/api/datasource/list] sending cat to client: " + string(jsonString))
		w.Write(jsonString)

	}).Methods("GET")

	r.HandleFunc("/api/store/cat/{store}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		store := vars["store"]
		fmt.Println("[/api/store/cat/{store}] store ", store)
		storeURL := "tcp://" + store + ":5555"
		storeHref := "https://" + store + ":8080"
		sc := coreStoreClient.New(request, &ac, "/run/secrets/ZMQ_PUBLIC_KEY", storeURL, false)
		storeCat, err := sc.GetStoreDataSourceCatalogue(storeHref)
		if err != nil {
			log.Err("[/api/store/cat/{store}] Error GetStoreDataSourceCatalogue " + err.Error())
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":` + err.Error() + `}`))
		}
		catStr, _ := json.Marshal(storeCat)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(catStr)

	}).Methods("GET")

	r.HandleFunc("/api/installed/list", func(w http.ResponseWriter, r *http.Request) {

		filters := filters.NewArgs()
		//filters.Add("label", "databox.type")
		services, _ := cli.ServiceList(context.Background(), types.ServiceListOptions{Filters: filters})

		res := []string{}
		for _, service := range services {
			res = append(res, service.Spec.Name)
			fmt.Println("[datasource/list] ", service.Spec.Name)
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			fmt.Println("error encoding json ", err)
		}

	}).Methods("GET")

	type listResult struct {
		Name         string          `json:"name"`
		Type         string          `json:"type"`
		DesiredState swarm.TaskState `json:"desiredState"`
		State        swarm.TaskState `json:"state"`
		Status       swarm.TaskState `json:"status"`
	}

	r.HandleFunc("/api/{type}/list", func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		serviceType := vars["type"]

		services, _ := cli.ServiceList(context.Background(), types.ServiceListOptions{})

		res := []listResult{}
		for _, service := range services {

			val, exists := service.Spec.Labels["databox.type"]
			if exists == false {
				//its not a databox service
				continue
			}
			if val != serviceType {
				//this is not the service were looking for
				continue
			}
			lr := listResult{
				Name: service.Spec.Name,
				Type: serviceType,
			}

			taskFilters := filters.NewArgs()
			taskFilters.Add("service", service.Spec.Name)
			tasks, _ := cli.TaskList(context.Background(), types.TaskListOptions{
				Filters: taskFilters,
			})
			if len(tasks) > 0 {
				latestTasks := tasks[0]
				latestTime := latestTasks.UpdatedAt

				for _, t := range tasks {
					if t.UpdatedAt.After(latestTime) {
						latestTasks = t
						latestTime = latestTasks.UpdatedAt
					}
				}

				lr.DesiredState = latestTasks.DesiredState
				lr.State = latestTasks.Status.State
				lr.Status = latestTasks.Status.State
			}

			res = append(res, lr)
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			fmt.Println("[/api/{type}/list] error encoding json ", err)
		}

	}).Methods("GET")

	r.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()
		slaString, _ := ioutil.ReadAll(r.Body)
		sla := databoxTypes.SLA{}
		err := json.Unmarshal(slaString, &sla)
		if err != nil {
			log.Err("[/api/install] Error invalid sla json " + err.Error() + "JSON=" + string(slaString))
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":` + err.Error() + `}`))
			return
		}

		log.Info("[/api/install] installing " + sla.Name)

		//add to proxy
		log.Debug("/api/install dboxproxy.Add " + sla.Name)
		dboxproxy.Add(sla.Name)

		//TODO check and return an error!!!
		log.Debug("/api/install LaunchFromSLA")
		cm.LaunchFromSLA(sla)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":200,"msg":"Success"}`))

		log.Debug("/api/install finished")

	}).Methods("POST")

	r.HandleFunc("/api/restart", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		bodyString, err := ioutil.ReadAll(r.Body)
		fmt.Println(bodyString)
		type jsonStruct struct {
			Name string `json:"id"`
		}
		jsonBody := jsonStruct{}
		err = json.Unmarshal(bodyString, &jsonBody)
		if err != nil {
			log.Err("[/api/restart] Malformed JSON " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg": "Malformed JSON"}`))
			return
		}
		if jsonBody.Name == "" {
			log.Err("[/api/restart] Missing container name ")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":Missing container name"}`))
			return
		}
		log.Info("[/api/restart] restarting - " + jsonBody.Name)

		err = cm.Restart(jsonBody.Name)
		if err != nil {
			log.Err("[/api/restart] Restrat " + jsonBody.Name + " failed. " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":"` + err.Error() + `"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":200,"msg":"Success"}`))
		return
	}).Methods("POST")

	r.HandleFunc("/api/uninstall", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		bodyString, err := ioutil.ReadAll(r.Body)
		fmt.Println(string(bodyString))
		type jsonStruct struct {
			Name string `json:"id"`
		}
		jsonBody := jsonStruct{}
		err = json.Unmarshal(bodyString, &jsonBody)
		if err != nil {
			log.Err("[/api/restart] Malformed JSON " + err.Error())
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":"Malformed JSON"}`))
			return
		}
		if jsonBody.Name == "" {
			log.Err("[/api/uninstall] Missing container name (id)")
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":Missing container name"}`))
			return
		}
		log.Info("[/api/uninstall] uninstalling " + jsonBody.Name)

		dboxproxy.Del(jsonBody.Name)

		err = cm.Uninstall(jsonBody.Name)
		if err != nil {
			log.Err("[/api/uninstall] Uninstall of " + jsonBody.Name + " failed. " + err.Error())
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"msg":"` + err.Error() + `"}`))
			return
		}

	}).Methods("POST")

	static := http.FileServer(http.Dir("./www/https"))
	r.PathPrefix("/").Handler(static)

	log.ChkErrFatal(http.ListenAndServeTLS(":443", "./certs/container-manager.pem", "./certs/container-manager.pem", r))
}