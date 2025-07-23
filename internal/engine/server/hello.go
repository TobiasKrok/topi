package server

import (
	"net/http"
)

func (e *EngineServer) helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}
