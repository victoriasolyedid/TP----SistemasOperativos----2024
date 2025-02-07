package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type Paquete struct {
	Valores []string `json:"valores"`
}

type ModuleHandler struct {
	RouteHandlers map[string]http.HandlerFunc
}

func RecibirPaquetes(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var paquete Paquete
	err := decoder.Decode(&paquete)
	if err != nil {
		fmt.Printf("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error al decodificar mensaje"))
		return
	}

	fmt.Println("me llego un paquete de un cliente")
	fmt.Printf("%+v\n", paquete)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirMensaje(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		fmt.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	
	fmt.Println("Me llego un mensaje de un cliente")
	fmt.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

/**
 * ServerStart: Inicia un servidor en el puerto especificado y con las rutas especificadas de ser necesario

 * @param port string
 * @param moduleRoutes optional
*/
func ServerStart(port int, moduleRoutes ...http.Handler) {
	mux := http.NewServeMux()

	mux.HandleFunc("/paquetes", RecibirPaquetes)
	mux.HandleFunc("/mensaje", RecibirMensaje)

	for _, route := range moduleRoutes {
		mux.Handle("/", route)
	}

	fmt.Printf("Server listening on port %d\n", port)
	err := http.ListenAndServe(":"+fmt.Sprintf("%v", port), mux)
	if err != nil {
		panic(err)
	}
}

/**
 * NewModule: Atiende la ruta de los módulos. Si no la encuentra, deja que el DefaultServeMux la atienda

 * @return ModuleHandler
*/
func (m *ModuleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, ok := m.RouteHandlers[r.Method+" "+r.URL.Path]
	if ok {
		handler(w, r)
		return
	}
	http.DefaultServeMux.ServeHTTP(w, r)
}