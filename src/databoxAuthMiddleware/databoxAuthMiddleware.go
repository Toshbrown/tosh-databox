package databoxAuthMiddleware

import (
	"crypto/rand"
	"databoxProxyMiddleware"
	log "databoxerrors"
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type databoxAuthMiddleware struct {
	sync.Mutex
	session  string
	proxy    *databoxProxyMiddleware.DataboxProxyMiddleware
	password string
}

func New(password string, proxy *databoxProxyMiddleware.DataboxProxyMiddleware) *databoxAuthMiddleware {

	return &databoxAuthMiddleware{
		password: password,
		session:  "",
		proxy:    proxy,
	}
}

func (d *databoxAuthMiddleware) AuthMiddleware(next http.Handler) http.Handler {

	Auth := func(w http.ResponseWriter, r *http.Request) {

		log.Debug(r.URL.Path)

		parts := strings.Split(r.URL.Path, "/")

		if len(parts) <= 2 {
			//call to / nothing to do here
			next.ServeHTTP(w, r)
			return
		}

		//TODO workout why calling d.proxy.Exists(parts[1]) hangs the https server ..... ???
		if parts[1] != "api" { //&& d.proxy.Exists(parts[1]) == false {
			//its not an api endpoint or a proxyed UI no need to auth
			log.Debug("Not UI or api call")
			next.ServeHTTP(w, r)
			return
		}

		d.Lock()
		defer d.Unlock()
		//handle connect request
		if len(parts) >= 3 && parts[2] == "connect" {
			//check password and issue session token if its OK
			log.Debug("Connect called checking password Token " + d.password + " = " + r.Header.Get("Authorization"))
			if ("Token " + d.password) == r.Header.Get("Authorization") {
				log.Debug("Password OK!")

				if d.session == "" {
					//make a new session token
					b := make([]byte, 24)
					rand.Read(b) //TODO This could error should check
					d.session = b64.StdEncoding.EncodeToString(b)
				}

				cookie := http.Cookie{Name: "databox_session", Value: d.session}
				http.SetCookie(w, &cookie)
				fmt.Fprintf(w, "connected")
				return
			}

			log.Err("Password validation error!")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Authorization Required")
			return
		}

		//Its not a connect request its an api call or a proxyed UI request
		// we must have a valid session cookie
		sessionCookie, _ := r.Cookie("databox_session")
		if sessionCookie != nil && d.session == sessionCookie.Value {
			log.Debug("Session cookie OK")
			//session cookie is ok continue to the next Middleware
			next.ServeHTTP(w, r)
			return
		}

		//if we get here were unauthorised
		log.Warn("Authorization failed")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Authorization Required")
	}

	return http.HandlerFunc(Auth)
}
