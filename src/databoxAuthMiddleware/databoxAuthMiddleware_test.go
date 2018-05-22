package databoxAuthMiddleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"databoxProxyMiddleware"

	"databoxAuthMiddleware"
)

func TestMain(m *testing.M) {
	Setup()
	retCode := m.Run()
	Teardown()
	os.Exit(retCode)
}

var auth *databoxAuthMiddleware.DataboxAuthMiddleware

func Setup() {

	proxy := databoxProxyMiddleware.New("")
	proxy.Add("arbiter")
	proxy.Add("some-random-app")
	auth = databoxAuthMiddleware.New("test_pass_word", proxy)
}

func Teardown() {
	//todo
}

func MockHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("MockHandler called")
	return
}

func TestAuthNoAuthHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/list", nil)
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

func TestAuthWrongAuthHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/list", nil)
	r.Header.Add("Authorization", "fishfinger")
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

var SessionCookies []*http.Cookie

func TestConnectAPICorrectAuthHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/connect", nil)
	r.Header.Add("Authorization", "Token test_pass_word")
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	SessionCookies = resp.Cookies()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}
func TestConnectAPIWrongAuthHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/connect", nil)
	r.Header.Add("Authorization", "fhfjdhsfkjdhsfk")
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

func TestCorrectSessionCookieAuth(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/app/list", nil)
	for _, c := range SessionCookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

func TestWrongSessionCookieAuth(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/app/list", nil)
	for _, c := range SessionCookies {
		if c.Name == "databox_session" {
			sc := c
			c.Value = "thisisnotasessioncookie"
			r.AddCookie(sc)
		} else {
			r.AddCookie(c)
		}
	}
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

func TestAuthCorrectAuthHeaderURLNotInProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/fishpie", nil)
	r.Header.Add("Authorization", "test_pass_word")
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

func TestAuthWrongAuthHeaderURLNotInProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/fishpie", nil)
	r.Header.Add("Authorization", "qwqwe")
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}

func TestWrongAuthHeaderURLInProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/arbiter/ui", nil)
	r.Header.Add("Authorization", "qwqwe")
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}
func TestCorrectAuthHeaderURLInProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/arbiter/ui", nil)
	for _, c := range SessionCookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	authHandlerFunc := auth.AuthMiddleware(http.HandlerFunc(MockHandler))
	authHandlerFunc.ServeHTTP(w, r)
	resp := w.Result()
	//body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Auth returned wrong status code: %d", resp.StatusCode)
	}
}
