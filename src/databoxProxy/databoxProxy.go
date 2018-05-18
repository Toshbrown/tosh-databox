package databoxProxy

import (
	"fmt"
	"io"
	"lib-go-databox/databoxRequest"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type databoxProxy struct {
	proxyList  map[string]string
	httpClient *http.Client
}

func New() *databoxProxy {

	h := databoxRequest.NewDataboxHTTPsAPI()

	return &databoxProxy{
		httpClient: h,
	}
}

func (d databoxProxy) Proxy(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(mux.Vars(r)["appurl"], "/")
	RequestURI := "https://" + parts[0] + ":8080/" + strings.Join(parts[1:], "/")

	fmt.Println("Proxying internal request to  ", RequestURI)

	req, err := http.NewRequest(r.Method, RequestURI, r.Body)
	for name, value := range r.Header {
		req.Header.Set(name, value[0])
	}
	resp, err := d.httpClient.Do(req)
	r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range resp.Header {
		w.Header().Set(k, v[0])
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()

}
