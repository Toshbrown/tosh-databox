package databoxAuthMiddleware

import (
	"containerManager/databoxProxyMiddleware"
	"crypto/rand"
	log "databoxlog"
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var allowedStaticPaths = map[string]string{"css": "", "js": "", "icons": "", "img": "", "": "", "cordova.js": ""}

type DataboxAuthMiddleware struct {
	sync.Mutex
	session  string //TODO this should be per user or at least per device
	proxy    *databoxProxyMiddleware.DataboxProxyMiddleware
	password string
	next     http.Handler
}

func New(password string, proxy *databoxProxyMiddleware.DataboxProxyMiddleware) *DataboxAuthMiddleware {

	return &DataboxAuthMiddleware{
		password: password,
		session:  "",
		proxy:    proxy,
	}
}

func (d *DataboxAuthMiddleware) AuthMiddleware(next http.Handler) http.Handler {

	auth := func(w http.ResponseWriter, r *http.Request) {

		//log.Debug("AuthMiddleware path=" + r.URL.Path)

		parts := strings.Split(r.URL.Path, "/")

		if _, ok := allowedStaticPaths[parts[1]]; ok {
			//its allowed no auth needed
			//log.Debug("its allowed no auth needed")
			next.ServeHTTP(w, r)
			return
		}

		d.Lock()
		defer d.Unlock()
		//handle connect request
		if len(parts) >= 3 && parts[2] == "connect" {
			//check password and issue session token if its OK
			log.Debug("Connect called checking password")
			if ("Token " + d.password) == r.Header.Get("Authorization") {
				log.Debug("Password OK!")
				if d.session == "" {
					//make a new session token
					b := make([]byte, 24)
					rand.Read(b) //TODO This could error should check
					d.session = b64.StdEncoding.EncodeToString(b)
				}

				cookie := http.Cookie{
					Name:   "session",
					Value:  d.session,
					Domain: r.URL.Hostname(),
					Path:   "/",
				}
				http.SetCookie(w, &cookie)
				fmt.Fprintf(w, "connected")
				return
			}

			log.Err("Password validation error!")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Authorization Required")
			return
		}

		//Its not a connect request
		// we must have a valid session cookie
		sessionCookie, _ := r.Cookie("session")
		if sessionCookie != nil && d.session == sessionCookie.Value {
			//log.Debug("Session cookie OK")
			//session cookie is ok continue to the next Middleware
			next.ServeHTTP(w, r)
			return
		}

		//if we get here were unauthorised
		log.Warn("Authorization failed")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Authorization Required")
		return
	}
	return http.HandlerFunc(auth)
}
