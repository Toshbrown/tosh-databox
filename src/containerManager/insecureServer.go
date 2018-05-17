package containerManager

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func ServeInsecure() {

	router := mux.NewRouter()
	static := http.FileServer(http.Dir("./www/http"))

	router.PathPrefix("/").Handler(static)

	log.Fatal(http.ListenAndServe(":80", router))
}
