package databoxProxyMiddleware

import (
	log "databoxerrors"
	"io"
	"lib-go-databox/databoxRequest"
	"net/http"
	"strings"
)

type databoxProxyMiddleware struct {
	proxyList  map[string]string
	httpClient *http.Client
	next       http.Handler
}

func New(rootCertPath string) *databoxProxyMiddleware {

	h := databoxRequest.NewDataboxHTTPsAPIWithPaths(rootCertPath)

	return &databoxProxyMiddleware{
		httpClient: h,
		proxyList:  make(map[string]string),
	}
}

func (d *databoxProxyMiddleware) ProxyMiddleware(next http.Handler) http.Handler {
	d.next = next
	return http.HandlerFunc(d.Proxy)
}

func (d databoxProxyMiddleware) Proxy(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(r.URL.Path, "/")

	if _, ok := d.proxyList[parts[1]]; ok == false {
		//no need to proxy
		d.next.ServeHTTP(w, r)
		return
	}

	RequestURI := "https://" + parts[1] + ":8080/" + strings.Join(parts[2:], "/")

	log.Info("Proxying internal request to  " + RequestURI)

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

func (d *databoxProxyMiddleware) Add(containerName string) {
	log.Info("[databoxProxyMiddleware.Add]" + containerName)
	d.proxyList[containerName] = containerName
}

func (d *databoxProxyMiddleware) Del(containerName string) {
	_, ok := d.proxyList[containerName]
	if ok {
		delete(d.proxyList, containerName)
	}
}
