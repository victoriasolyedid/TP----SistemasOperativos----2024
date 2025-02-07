package kernel_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	"github.com/sisoputnfrba/tp-golang/utils/device"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
	"github.com/sisoputnfrba/tp-golang/utils/slice"
)

/**
 * GetIOInterface: Recibe una interfaz de IO y la agrega al sistema.

 * @param w: http.ResponseWriter -> Respuesta a enviar.
 * @param r: *http.Request -> Request recibido.
 */
func GetIOInterface(w http.ResponseWriter, r *http.Request) {
	var interf device.T_IOInterface

	err := json.NewDecoder(r.Body).Decode(&interf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slice.Push(&globals.Interfaces, interf)

	fmt.Printf("Interface received, type: %s, port: %d\n", interf.InterfaceType, interf.InterfacePort)

	w.WriteHeader(http.StatusOK)
}

type SearchInterface struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

/**
 * ExisteInterfaz: Verifica si una interfaz existe en el sistema.

 * @param w: http.ResponseWriter -> Respuesta a enviar.
 * @param r: *http.Request -> Request recibido.
*/

func ExisteInterfaz(w http.ResponseWriter, r *http.Request) {
	var received_data SearchInterface
	err := json.NewDecoder(r.Body).Decode(&received_data)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}

	fmt.Printf("Received data: %s, %s\n", received_data.Name, received_data.Type)

	aux, err := SearchDeviceByName(received_data.Name)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
	}

	var response bool
	if aux.InterfaceType == received_data.Type {
		response = true
	} else {
		response = false
	}

	jsonResp, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

/**
 * SearchDeviceByName: Busca una interfaz por su nombre.

 * @param deviceName: string -> Nombre de la interfaz a buscar.
 * @return device.T_IOInterface -> Interfaz encontrada.
*/

func SearchDeviceByName(deviceName string) (device.T_IOInterface, error) {
	for _, interf := range globals.Interfaces {
		if interf.InterfaceName == deviceName {
			fmt.Println("Interfaz encontrada: ", interf)
			return interf, nil
		}
	}
	return device.T_IOInterface{}, fmt.Errorf("device not found")
}

// * Types para realizar solicitudes a IO
type GenSleep struct {
	Pcb         pcb.T_PCB
	Inter       device.T_IOInterface
	TimeToSleep int
}

type StdinRead struct {
	Pcb                pcb.T_PCB
	Inter              device.T_IOInterface
	DireccionesFisicas []globals.DireccionTamanio
}

type StdoutWrite struct {
	Pcb                pcb.T_PCB
	Inter              device.T_IOInterface
	DireccionesFisicas []globals.DireccionTamanio
}

type DialFSRequest struct {
	Pcb           pcb.T_PCB
	Inter         device.T_IOInterface
	NombreArchivo string
	Tamanio       int
	Puntero       int
	Direccion     []globals.DireccionTamanio
	Operacion     string
}

/**
 * SolicitarGenSleep: Solicita a IO la operación de GEN_SLEEP.

 * @param pcb: pcb.T_PCB -> PCB que solicita la operación.
*/

func SolicitarGenSleep(pcb pcb.T_PCB) {
	genSleepDataDecoded := genericInterfaceBody.(struct {
		InterfaceName string
		SleepTime     int
	})

	newInter, err := SearchDeviceByName(genSleepDataDecoded.InterfaceName)
	if err != nil {
		fmt.Printf("Device not found: %v", err)
	}

	genSleep := GenSleep{
		Pcb:         pcb,
		Inter:       newInter,
		TimeToSleep: genSleepDataDecoded.SleepTime,
	}

	globals.EnganiaPichangaMutex.Unlock()

	jsonData, err := json.Marshal(genSleep)
	if err != nil {
		fmt.Printf("Failed to encode GenSleep request: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/io-operate", newInter.InterfaceIP, newInter.InterfacePort)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send PCB: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected response status: %s", resp.Status)
	}
}

/**
 * SolicitarStdinRead: Solicita a IO la operación de STDIN_READ.

 * @param pcb: pcb.T_PCB -> PCB que solicita la operación.
*/

func SolicitarStdinRead(pcb pcb.T_PCB) {
	stdinDataDecoded := genericInterfaceBody.(struct {
		DireccionesFisicas []globals.DireccionTamanio
		InterfaceName      string
		Tamanio            int
	})

	fmt.Println("RECIBE STDIN READ: ", stdinDataDecoded)

	newInter, err := SearchDeviceByName(stdinDataDecoded.InterfaceName)
	if err != nil {
		fmt.Printf("Device not found: %v", err)
	}

	stdinRead := StdinRead{
		Pcb:                pcb,
		Inter:              newInter,
		DireccionesFisicas: stdinDataDecoded.DireccionesFisicas,
	}

	fmt.Println("LE QUIERE MANDAR A IO: ", stdinRead)

	globals.EnganiaPichangaMutex.Unlock()

	jsonData, err := json.Marshal(stdinRead)
	if err != nil {
		fmt.Printf("Failed to encode StdinRead request: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/io-operate", newInter.InterfaceIP, newInter.InterfacePort)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send PCB: %v", err)
	}

	fmt.Println("IO STDIN FUE AVISADO POR KERNEL")

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected response status: %s", resp.Status)
	}
}

/**
 * SolicitarStdoutWrite: Solicita a IO la operación de STDOUT_WRITE.

 * @param pcb: pcb.T_PCB -> PCB que solicita la operación.
*/

func SolicitarStdoutWrite(pcb pcb.T_PCB) {
	stdoutDataDecoded := genericInterfaceBody.(struct {
		DireccionesFisicas []globals.DireccionTamanio
		InterfaceName      string
	})

	newInter, err := SearchDeviceByName(stdoutDataDecoded.InterfaceName)
	if err != nil {
		fmt.Printf("Device not found: %v", err)
	}

	stdoutWrite := StdoutWrite{
		Pcb:                pcb,
		Inter:              newInter,
		DireccionesFisicas: stdoutDataDecoded.DireccionesFisicas,
	}

	globals.EnganiaPichangaMutex.Unlock()

	jsonData, err := json.Marshal(stdoutWrite)
	if err != nil {
		fmt.Printf("Failed to encode StdoutWrite request: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/io-operate", newInter.InterfaceIP, newInter.InterfacePort)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send PCB: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected response status: %s", resp.Status)
	}
}

/**
 * SolicitarDialFS: Solicita a IO la operación de DIAL_FS.

 * @param pcb: pcb.T_PCB -> PCB que solicita la operación.
*/

func SolicitarDialFS(pcb pcb.T_PCB) {
	dialFsDataDecoded := genericInterfaceBody.(struct {
		InterfaceName string
		FileName      string
		Size          int
		Pointer       int
		Address       []globals.DireccionTamanio
		Operation     string
	})

	newInter, err := SearchDeviceByName(dialFsDataDecoded.InterfaceName)
	if err != nil {
		fmt.Printf("Device not found: %v", err)
	}

	dialFS := DialFSRequest{
		Pcb:           pcb,
		Inter:         newInter,
		NombreArchivo: dialFsDataDecoded.FileName,
		Tamanio:       dialFsDataDecoded.Size,
		Puntero:       dialFsDataDecoded.Pointer,
		Direccion:     dialFsDataDecoded.Address,
		Operacion:     dialFsDataDecoded.Operation,
	}

	globals.EnganiaPichangaMutex.Unlock()

	jsonData, err := json.Marshal(dialFS)
	if err != nil {
		fmt.Printf("Failed to encode DialFS request: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/io-operate", newInter.InterfaceIP, newInter.InterfacePort)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send PCB: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected response status: %s", resp.Status)
	}
}

var genericInterfaceBody interface{}

/**
 * RecvData_gensleep: Recibe desde CPU la información necesaria para solicitar un GEN_SLEEP. 

 * @param w: http.ResponseWriter -> Respuesta a enviar.
 * @param r: http.Request -> Request recibido.
*/
func RecvData_gensleep(w http.ResponseWriter, r *http.Request) {
	var received_data struct {
		InterfaceName string
		SleepTime     int
	}

	err := json.NewDecoder(r.Body).Decode(&received_data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	genericInterfaceBody = received_data

	w.WriteHeader(http.StatusOK)
}

/**
 * RecvData_stdin: Recibe desde CPU la información necesaria para solicitar un STDIN_READ.

 * @param w: http.ResponseWriter -> Respuesta a enviar.
 * @param r: http.Request -> Request recibido.
*/

func RecvData_stdin(w http.ResponseWriter, r *http.Request) {
	var received_data struct {
		DireccionesFisicas []globals.DireccionTamanio
		InterfaceName      string
		Tamanio            int
	}

	err := json.NewDecoder(r.Body).Decode(&received_data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("Received data: ", received_data)
	genericInterfaceBody = received_data

	w.WriteHeader(http.StatusOK)
}

/**
 * RecvData_stdout: Recibe desde CPU la información necesaria para solicitar un STDOUT_WRITE.

 * @param w: http.ResponseWriter -> Respuesta a enviar.
 * @param r: *http.Request -> Request recibido.
*/
func RecvData_stdout(w http.ResponseWriter, r *http.Request) {
	var received_data struct {
		DireccionesFisicas []globals.DireccionTamanio
		InterfaceName      string
	}

	err := json.NewDecoder(r.Body).Decode(&received_data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	genericInterfaceBody = received_data

	w.WriteHeader(http.StatusOK)
}

/*
	 RecvData_dialfs: Recibe desde CPU la información necesaria para solicitar un DIAL_FS.

	 @params:
		- w: http.ResponseWriter -> Respuesta a enviar.
		- r: *http.Request -> Request recibido.
*/

func RecvData_dialfs(w http.ResponseWriter, r *http.Request) {
	var received_data struct {
		//	Pcb 					pcb.T_PCB
		InterfaceName string
		FileName      string
		Size          int
		Pointer       int
		Address       []globals.DireccionTamanio
		Operation     string
	}

	err := json.NewDecoder(r.Body).Decode(&received_data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	genericInterfaceBody = received_data

	w.WriteHeader(http.StatusOK)
}

/*
	RecvPCB_IO: Recibe el PCB bloqueado por IO, lo desbloquea y lo agrega a la cola de STS.

	@params:
		- w: http.ResponseWriter -> Respuesta a enviar.
		- r: *http.Request -> Request recibido.

*/
func RecvPCB_IO(w http.ResponseWriter, r *http.Request) {
	var received_pcb pcb.T_PCB

	err := json.NewDecoder(r.Body).Decode(&received_pcb)
	if err != nil {
		http.Error(w, "Failed to decode PCB", http.StatusBadRequest)
		return
	}

	fmt.Println("PCB que nos manda IO (Kernel): PC: ", received_pcb.PC, "PID: ", received_pcb.PID)

	fmt.Println("Blocked: ", globals.Blocked)

	RemoveByID(received_pcb.PID)
	globals.ChangeState(&received_pcb, "READY")

	if (received_pcb.Quantum != globals.Configkernel.Quantum) {
		slice.Push(&globals.STS_Priority, received_pcb)
	} else {
		slice.Push(&globals.STS, received_pcb)
	}

	globals.STSCounter <- int(received_pcb.PID)


	w.WriteHeader(http.StatusOK)
}
