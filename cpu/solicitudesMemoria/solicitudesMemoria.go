package solicitudesmemoria

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
)

// Peticion para RESIZE de memoria (DESDE CPU A MEMORIA)
func Resize(tamanio int) string {
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/resize", globals.Configcpu.IP_memory, globals.Configcpu.Port_memory)
	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		return "error"
	}

	q := req.URL.Query()
	tamanioEnString := strconv.Itoa(tamanio)
	q.Add("tamanio", tamanioEnString)
	q.Add("pid", strconv.Itoa(int(globals.CurrentJob.PID)))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		return "error"
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		return "Error al realizar la petición de resize"
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return "error"
	}
	//En caso de que la respuesta de la memoria sea Out of Memory, se deberá devolver el contexto de ejecución al Kernel informando de esta situación
	// Y Avisar que el error es por out of memory
	return string(bodyBytes)
}

type BodyRequestEscribir struct {
	DireccionesTamanios []globals.DireccionTamanio `json:"direcciones_tamanios"`
	Valor_a_escribir    []byte                     `json:"valor_a_escribir"`
	Pid                 int                        `json:"pid"`
}

func SolicitarEscritura(direccionesTamanios []globals.DireccionTamanio, valorAEscribir []byte, pid int) {
	body, err := json.Marshal(BodyRequestEscribir{
		DireccionesTamanios: direccionesTamanios,
		Valor_a_escribir:    valorAEscribir,
		Pid:                 pid,
	})
	if err != nil {
		return
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/write", globals.Configcpu.IP_memory, globals.Configcpu.Port_memory)
	escribirEnMemoria, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	escribirEnMemoria.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(escribirEnMemoria)
	if err != nil {
		fmt.Println("error")
	}

	if respuesta.StatusCode != http.StatusOK {
		fmt.Println("Error al realizar la escritura")
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		fmt.Println("error")
	}

	// La respuesta puede ser un "Ok" o u "Error: dirección o tamanio fuera de rango"
	respuestaEnString := string(bodyBytes)

	respuestaSinComillas := strings.Trim(respuestaEnString, `"`)

	fmt.Println("Respuesta de memoria: ", respuestaSinComillas)
	fmt.Println("Valor a escribir bytes: ", valorAEscribir)
	valorAEscribirEnString := string(valorAEscribir)
	fmt.Println("Valor a escribir en string: ", valorAEscribirEnString)

	if respuestaSinComillas != "OK" {
		fmt.Println("Se produjo un error al escribir", respuestaSinComillas)
	} else {
		for _, df := range direccionesTamanios {
			cantEscrita := 0
			fmt.Println("Valor a escribir: ", valorAEscribir)
			log.Printf("PID: %d - Acción: ESCRIBIR - Dirección Física: %d - Valor: %b", pid, df.DireccionFisica, valorAEscribir[cantEscrita:df.Tamanio])
			cantEscrita += df.Tamanio
		}
	}
}

type BodyRequestLeer struct {
	DireccionesTamanios []globals.DireccionTamanio `json:"direcciones_tamanios"`
	Pid                 int                        `json:"pid"`
}

type BodyADevolver struct {
	Contenido [][]byte `json:"contenido"`
}

// LE SOLICITO A MEMORIA LEER Y DEVOLVER LO QUE ESTÉ EN LA DIREC FISICA INDICADA
func SolicitarLectura(direccionesFisicas []globals.DireccionTamanio, pid int) []byte {
	var bodyResponseLeer BodyADevolver

	jsonDirecYTamanio, err := json.Marshal(BodyRequestLeer{
		DireccionesTamanios: direccionesFisicas,
		Pid:                 pid,
	})
	if err != nil {
		return []byte("error")
	}

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/read", globals.Configcpu.IP_memory, globals.Configcpu.Port_memory)
	leerMemoria, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonDirecYTamanio))
	if err != nil {
		return []byte("error")
	}

	fmt.Println("Solicito lectura de memoria", jsonDirecYTamanio)

	leerMemoria.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(leerMemoria)
	if err != nil {
		return []byte("error")
	}

	if respuesta.StatusCode != http.StatusOK {
		return []byte("Error al realizar la lectura")
	}

	err = json.NewDecoder(respuesta.Body).Decode(&bodyResponseLeer)
	if err != nil {
		return []byte("error al deserializar la respuesta")
	}
	fmt.Println("Direcciones fisicas: ", direccionesFisicas)

	for i, df := range direccionesFisicas {
		contenido := bodyResponseLeer.Contenido[i]
		fmt.Println("Contenido leido: ", contenido)
		log.Printf("PID: %d - Acción: LEER - Dirección Física: %d - Valor: %b", pid, df.DireccionFisica, contenido)
	}

	var bytesConcatenados []byte
	for _, sliceBytes := range bodyResponseLeer.Contenido {
		bytesConcatenados = append(bytesConcatenados, sliceBytes...)
	}
	return bytesConcatenados
}
