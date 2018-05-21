package databoxProxyMiddleware

import (
	log "databoxerrors"
	"io"
	"lib-go-databox/databoxRequest"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type DataboxProxyMiddleware struct {
	sync.Mutex
	proxyList  map[string]string
	httpClient *http.Client
	next       http.Handler
}

func New(rootCertPath string) *DataboxProxyMiddleware {

	h := databoxRequest.NewDataboxHTTPsAPIWithPaths(rootCertPath)

	return &DataboxProxyMiddleware{
		httpClient: h,
		proxyList:  make(map[string]string),
	}
}

func (d *DataboxProxyMiddleware) ProxyMiddleware(next http.Handler) http.Handler {
	d.next = next
	return http.HandlerFunc(d.Proxy)
}

func (d *DataboxProxyMiddleware) Proxy(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(r.URL.Path, "/")

	d.Lock()
	defer d.Unlock()
	if _, ok := d.proxyList[parts[1]]; ok == false {
		//no need to proxy
		d.next.ServeHTTP(w, r)
		return
	}

	RequestURI := "https://" + parts[1] + ":8080/" + strings.Join(parts[2:], "/")

	log.Debug("Proxying internal request to  " + RequestURI)

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
	return
}

func (d *DataboxProxyMiddleware) Add(containerName string) {
	d.Lock()
	defer d.Unlock()
	log.Debug("[databoxProxyMiddleware.Add] " + containerName)
	d.proxyList[containerName] = containerName
}

func (d *DataboxProxyMiddleware) Del(containerName string) {
	d.Lock()
	defer d.Unlock()
	_, ok := d.proxyList[containerName]
	if ok {
		delete(d.proxyList, containerName)
	}
}

func (d *DataboxProxyMiddleware) Exists(containerName string) bool {
	log.Debug("DataboxProxyMiddleware.Exists called for " + containerName)
	d.Lock()
	defer d.Unlock()
	_, ok := d.proxyList[containerName]
	log.Debug("DataboxProxyMiddleware.Exists returning " + strconv.FormatBool(ok))
	return ok
}
