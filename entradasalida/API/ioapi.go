package IO_api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sisoputnfrba/tp-golang/entradasalida/globals"
	"github.com/sisoputnfrba/tp-golang/utils/device"
	"github.com/sisoputnfrba/tp-golang/utils/generics"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
)

type CantUnidadesTrabajo struct {
	Unidades int `json:"cantUnidades"`
}

func HandshakeKernel(nombre string) error {
	genInterface := device.T_IOInterface{
		InterfaceName: nombre,
		InterfaceType: globals.ConfigIO.Type,
		InterfaceIP:   globals.ConfigIO.Ip,
		InterfacePort: globals.ConfigIO.Port,
	}

	jsonData, err := json.Marshal(genInterface)
	if err != nil {
		return fmt.Errorf("failed to encode interface: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/io-handshake", globals.ConfigIO.Ip_kernel, globals.ConfigIO.Port_kernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("POST request failed. Failed to send interface: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	fmt.Println("Handshake con Kernel exitoso")

	return nil
}

// Hay que declarar los tipos de body que se van a recibir desde kernel porque por alguna razón no se puede crear un struct type dentro de una función con un tipo creado por uno mismo, están todos en globals

func InterfaceQueuePCB(w http.ResponseWriter, r *http.Request) {
	switch globals.ConfigIO.Type {
	case "GENERICA":
		var decodedStruct globals.GenSleep

		err := json.NewDecoder(r.Body).Decode(&decodedStruct)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		fmt.Println("Nueva PCB ID: ", decodedStruct.Pcb.PID, " para usar Interfaz")
		globals.Generic_QueueChannel <- decodedStruct

	case "STDIN":
		var decodedStruct globals.StdinRead

		err := json.NewDecoder(r.Body).Decode(&decodedStruct)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		fmt.Println("Nueva PCB ID: ", decodedStruct.Pcb.PID, " para usar Interfaz")
		globals.Stdin_QueueChannel <- decodedStruct

	case "STDOUT":
		var decodedStruct globals.StdoutWrite

		err := json.NewDecoder(r.Body).Decode(&decodedStruct)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		fmt.Println("Nueva PCB ID: ", decodedStruct.Pcb.PID, " para usar Interfaz")
		globals.Stdout_QueueChannel <- decodedStruct

	case "DIALFS":
		var decodedStruct globals.DialFSRequest

		err := json.NewDecoder(r.Body).Decode(&decodedStruct)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		fmt.Println("Nueva PCB ID: ", decodedStruct.Pcb.PID, " para usar Interfaz")
		globals.DialFS_QueueChannel <- decodedStruct
	}

	w.WriteHeader(http.StatusOK)
}

func IOWork() {
	switch globals.ConfigIO.Type {
	case "GENERICA":
		var interfaceToWork globals.GenSleep
		for {
			interfaceToWork = <-globals.Generic_QueueChannel

			IO_GEN_SLEEP(interfaceToWork.TimeToSleep, interfaceToWork.Pcb)
			fmt.Println("Fin de bloqueo para el PID: ", interfaceToWork.Pcb.PID)
			returnPCB(interfaceToWork.Pcb)
		}
	case "STDIN":
		var interfaceToWork globals.StdinRead
		for {
			interfaceToWork = <-globals.Stdin_QueueChannel

			IO_STDIN_READ(interfaceToWork.Pcb, interfaceToWork.DireccionesFisicas)
			fmt.Println("Fin de bloqueo para el PID: ", interfaceToWork.Pcb.PID)
			returnPCB(interfaceToWork.Pcb)
		}
	case "STDOUT":
		var interfaceToWork globals.StdoutWrite
		for {
			interfaceToWork = <-globals.Stdout_QueueChannel

			IO_STDOUT_WRITE(interfaceToWork.Pcb, interfaceToWork.DireccionesFisicas)
			fmt.Println("Fin de bloqueo para el PID: ", interfaceToWork.Pcb.PID)
			returnPCB(interfaceToWork.Pcb)
		}

	case "DIALFS":
		var interfaceToWork globals.DialFSRequest
		for {
			interfaceToWork = <-globals.DialFS_QueueChannel
			time.Sleep(time.Duration(globals.ConfigIO.Unit_work_time) * time.Millisecond) //! agrego esto
			IO_DIALFS(interfaceToWork)
			fmt.Println("Fin de bloqueo para el PID: ", interfaceToWork.Pcb.PID)
			returnPCB(interfaceToWork.Pcb)

		}
	}
}

func returnPCB(pcb pcb.T_PCB) {
	generics.DoRequest("POST", fmt.Sprintf("http://%s:%d/io-return-pcb", globals.ConfigIO.Ip_kernel, globals.ConfigIO.Port_kernel), pcb, nil)
}

// ------------------------- OPERACIONES -------------------------

func IO_GEN_SLEEP(sleepTime int, pcb pcb.T_PCB) {
	sleepTimeTotal := time.Duration(sleepTime*globals.ConfigIO.Unit_work_time) * time.Millisecond
	log.Printf("PID: %d - Operacion: IO_GEN_SLEEP", pcb.PID)
	time.Sleep(sleepTimeTotal)
}

func IO_STDIN_READ(pcb pcb.T_PCB, direccionesFisicas []globals.DireccionTamanio) {
	// Lee datos de la entrada
	log.Printf("PID: %d - Operacion: IO_STDIN_READ", pcb.PID)
	fmt.Print("Ingrese datos: ")
	reader := bufio.NewReader(os.Stdin)
	data, _ := reader.ReadString('\n')

	// Le pido a memoria que me guarde los datos
	url := fmt.Sprintf("http://%s:%d/write", globals.ConfigIO.Ip_memory, globals.ConfigIO.Port_memory)

	bodyWrite, err := json.Marshal(struct {
		DireccionesTamanios []globals.DireccionTamanio `json:"direcciones_tamanios"`
		Valor_a_escribir    []byte                     `json:"valor_a_escribir"`
		Pid                 int                        `json:"pid"`
	}{direccionesFisicas, []byte(data), int(pcb.PID)})
	if err != nil {
		fmt.Println("Failed to encode data: ", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(bodyWrite))
	if err != nil {
		fmt.Println("Failed to send data: ", err)
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("Unexpected response status: ", response.Status)
	}
}

type BodyRequestLeer struct {
	DireccionesTamanios []globals.DireccionTamanio `json:"direcciones_tamanios"`
	Pid                 int                        `json:"pid"`
}
type BodyADevolver struct {
	Contenido [][]byte `json:"contenido"`
}

func IO_STDOUT_WRITE(pcb pcb.T_PCB, direccionesFisicas []globals.DireccionTamanio) {
	log.Printf("PID: %d - Operacion: IO_STDOUT_WRITE", pcb.PID)
	url := fmt.Sprintf("http://%s:%d/read", globals.ConfigIO.Ip_memory, globals.ConfigIO.Port_memory)

	bodyRead, err := json.Marshal(BodyRequestLeer{
		DireccionesTamanios: direccionesFisicas,
		Pid:                 int(pcb.PID),
	})
	if err != nil {
		return
	}

	datosLeidos, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRead))
	if err != nil {
		fmt.Println("Failed to receive data: ", err)
	}

	if datosLeidos.StatusCode != http.StatusOK {
		fmt.Println("Unexpected response status: ", datosLeidos.Status)
	}

	var response BodyADevolver

	err = json.NewDecoder(datosLeidos.Body).Decode(&response)
	if err != nil {
		return
	}

	var bytesConcatenados []byte
	for _, sliceBytes := range response.Contenido {
		bytesConcatenados = append(bytesConcatenados, sliceBytes...)
	}

	// Convierto los datos a string
	responseString := string(bytesConcatenados)

	// Consumo una unidad de trabajo
	time.Sleep(time.Duration(globals.ConfigIO.Unit_work_time) * time.Millisecond)

	fmt.Print("Datos leidos: *")
	// Escribo los datos en la salida (los muestro por pantalla)
	writer := bufio.NewWriter(os.Stdout)
	writer.WriteString(responseString)
	writer.Flush()
	fmt.Print("*\n")
}

// ------------------------- DIALFS -------------------------

func IO_DIALFS(interfaceToWork globals.DialFSRequest) {
	pid := int(interfaceToWork.Pcb.PID)
	nombreArchivo := interfaceToWork.NombreArchivo

	switch interfaceToWork.Operacion {
	case "CREATE":
		CreateFile(pid, nombreArchivo)

	case "DELETE":
		DeleteFile(pid, nombreArchivo)

	case "READ":
		ReadFile(pid, nombreArchivo, interfaceToWork.Direccion, interfaceToWork.Tamanio, interfaceToWork.Puntero)

	case "WRITE":
		WriteFile(pid, nombreArchivo, interfaceToWork.Direccion, interfaceToWork.Tamanio, interfaceToWork.Puntero)

	case "TRUNCATE":
		TruncateFile(pid, nombreArchivo, interfaceToWork.Tamanio)
	}

	fmt.Println("Operación ", interfaceToWork.Operacion, " finalizada - Archivo: ", nombreArchivo)
	fmt.Println("El archivo de bloques.dat es: ", globals.Blocks)
	fmt.Println("El archivo de bitmap.dat es: ", globals.CurrentBitMap)
}

func IO_DIALFS_READ(pid int, direccionesFisicas []globals.DireccionTamanio, contenido []byte) {
	// Le pido a memoria que me guarde los datos
	url := fmt.Sprintf("http://%s:%d/write", globals.ConfigIO.Ip_memory, globals.ConfigIO.Port_memory)

	bodyWrite, err := json.Marshal(struct {
		DireccionesTamanios []globals.DireccionTamanio `json:"direcciones_tamanios"`
		Valor_a_escribir    []byte                     `json:"valor_a_escribir"`
		Pid                 int                        `json:"pid"`
	}{direccionesFisicas, contenido, pid})
	if err != nil {
		fmt.Println("Failed to encode data: ", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(bodyWrite))
	if err != nil {
		fmt.Println("Failed to send data: ", err)
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("Unexpected response status: ", response.Status)
	}

}

func IO_DIALFS_WRITE(pid int, direccionesFisicas []globals.DireccionTamanio) []byte {

	url := fmt.Sprintf("http://%s:%d/read", globals.ConfigIO.Ip_memory, globals.ConfigIO.Port_memory)

	bodyRead, err := json.Marshal(BodyRequestLeer{
		DireccionesTamanios: direccionesFisicas,
		Pid:                 pid,
	})
	if err != nil {
		return nil
	}

	datosLeidos, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRead))
	if err != nil {
		fmt.Println("Failed to receive data: ", err)
	}

	if datosLeidos.StatusCode != http.StatusOK {
		fmt.Println("Unexpected response status: ", datosLeidos.Status)
	}

	var response BodyADevolver
	err = json.NewDecoder(datosLeidos.Body).Decode(&response)
	if err != nil {
		return []byte("error al deserializar la respuesta")
	}

	var bytesConcatenados []byte
	for _, sliceBytes := range response.Contenido {
		bytesConcatenados = append(bytesConcatenados, sliceBytes...)
	}
	// Consumo una unidad de trabajo
	return bytesConcatenados
}
