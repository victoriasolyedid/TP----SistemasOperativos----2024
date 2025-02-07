package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type Paquete struct {
	Valores []string `json:"valores"`
}

func EnviarMensaje(ip string, puerto int, mensajeTxt string) {
	mensaje := Mensaje{Mensaje: mensajeTxt}
	body, err := json.Marshal(mensaje)
	if err != nil {
		fmt.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/mensaje", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	fmt.Printf("respuesta del servidor: %s", resp.Status)
}

func EnviarPaquete(ip string, puerto int, paquete Paquete) {
	body, err := json.Marshal(paquete)
	if err != nil {
		fmt.Printf("error codificando mensajes: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/paquetes", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("error enviando mensajes a ip: %s puerto: %d", ip, puerto)
	}

	fmt.Printf("respuesta del servidor: %s", resp.Status)
}

func GenerarYEnviarPaquete(ipdestino string, puertodestino int) {
	paquete := Paquete{}

	// Leemos y cargamos el paquete
	fmt.Println("Cargando Paquete. Ingrese los valores")
	reader := bufio.NewReader(os.Stdin)
	i := 0
	for i < 1 {
		text, _ := reader.ReadString('\n')
		if text == "\n" {
			i++
		}
		paquete.Valores = append(paquete.Valores, text)
	}

	fmt.Printf("paqute a enviar:\n %+v", paquete)
	EnviarPaquete(ipdestino, puertodestino, paquete)
}