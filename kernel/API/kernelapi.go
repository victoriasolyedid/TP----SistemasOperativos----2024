package kernel_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	resource "github.com/sisoputnfrba/tp-golang/kernel/resources"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
	"github.com/sisoputnfrba/tp-golang/utils/slice"
)

// ! Verificar que no se genere ningún problema de dependencias con resources

/* Glossary:
- BRQ: Body Request
- BRS: Body Response
*/

type ProcessStart_BRQ struct {
	PID  uint32 `json:"pid"`
	Path string `json:"path"`
}

type ProcessStart_BRS struct {
	PID uint32 `json:"pid"`
}

type GetInstructions_BRQ struct {
	Path string `json:"path"`
	Pid  uint32 `json:"pid"`
	Pc   uint32 `json:"pc"`
}

/**
  - ProcessInit: Inicia un proceso en base a un archivo dentro del FS de Linux.
*/
func ProcessInit(w http.ResponseWriter, r *http.Request) {
	var request ProcessStart_BRQ
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pathInst, err := json.Marshal(fmt.Sprintf(request.Path))
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	pathInstString := string(pathInst)

	newPcb := &pcb.T_PCB{
		PID:     request.PID, // ! ESTO NO ESTABA >:v
		PC:      0,
		Quantum: globals.Configkernel.Quantum,
		CPU_reg: map[string]interface{}{
			"AX":  uint8(0),
			"BX":  uint8(0),
			"CX":  uint8(0),
			"DX":  uint8(0),
			"EAX": uint32(0),
			"EBX": uint32(0),
			"ECX": uint32(0),
			"EDX": uint32(0),
			"SI":  uint32(0),
			"DI":  uint32(0),
			"PC":  uint32(0),
		},
		State:             "NEW",
		EvictionReason:    "",
		Resources:         make(map[string]int), // * El valor por defecto es 0, tener en cuenta por las dudas a la hora de testear
		RequestedResource: "",
		Executions:        0,
	}

	var respBody ProcessStart_BRS = ProcessStart_BRS{PID: newPcb.PID}
	response, err := json.Marshal(respBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtengo las instrucciones del proceso
	url := fmt.Sprintf("http://%s:%d/instrucciones", globals.Configkernel.IP_memory, globals.Configkernel.Port_memory)

	bodyInst, err := json.Marshal(GetInstructions_BRQ{
		Path: pathInstString,
		Pid:  newPcb.PID,
		Pc:   newPcb.PC,
	})
	if err != nil {
		return
	}

	requerirInstrucciones, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyInst))
	if err != nil {
		fmt.Printf("POST request failed (No se pueden cargar instrucciones): %v", err)
	}

	cliente := &http.Client{}
	requerirInstrucciones.Header.Set("Content-Type", "application/json")
	recibirRespuestaInstrucciones, err := cliente.Do(requerirInstrucciones)
	if err != nil || recibirRespuestaInstrucciones.StatusCode != http.StatusOK {
		fmt.Println("Error en CargarInstrucciones (memoria)", err)
	}

	// Si la lista está vacía, la desbloqueo
	if len(globals.LTS) == 0 {
		globals.LTSMutex.Lock()
		slice.Push(&globals.LTS, *newPcb)
		defer globals.LTSMutex.Unlock()
		<-globals.EmptiedList
	} else {
		globals.LTSMutex.Lock()
		slice.Push(&globals.LTS, *newPcb)
		defer globals.LTSMutex.Unlock()
	}

	log.Printf("Se crea el proceso %d en %s\n", newPcb.PID, newPcb.State)
	
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

/**
  - ProcessDelete: Elimina un proceso en base a un PID. Realiza las operaciones como si el proceso llegase a EXIT
*/
func ProcessDelete(w http.ResponseWriter, r *http.Request) {
	pidString := r.PathValue("pid")
	pid, err := GetPIDFromString(pidString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Si el proceso está en ejecución, se envía una interrupción para desalojarlo con INTERRUPTED_BY_USER, de lo contrario se elimina directamente y se saca de la cola en la que se encuentre 
	if (pid == globals.CurrentJob.PID && globals.CurrentJob.State == "EXEC") {
		SendInterrupt("DELETE", pid, -1)
	} else {
		DeleteByID(pid)
	}

	w.WriteHeader(http.StatusOK)
}

type ProcessStatus_BRS struct {
	State string `json:"state"`
}

/**
  - ProcessState: Devuelve el estado de un proceso en base a un PID
*/
func ProcessState(w http.ResponseWriter, r *http.Request) {
	pidString := r.PathValue("pid")
	pid, err := GetPIDFromString(pidString)
	if err != nil {
		fmt.Println("Error al convertir PID a string: ", pidString, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	process, _ := SearchByID(pid, getProcessList())
	if process == nil {
		http.Error(w, "Process not found", http.StatusNotFound)
		return
	}

	result := ProcessStatus_BRS{State: process.State}

	response, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

/**
 * PlanificationStart: Retoma el STS y LTS en caso de que la planificación se encuentre pausada. Si no, ignora la petición.
 */
func PlanificationStart(w http.ResponseWriter, r *http.Request) {
	globals.PlanningState = "RUNNING"
	<- globals.LTSPlanBinary
	<- globals.STSPlanBinary
	fmt.Println("Planification Started")
	w.WriteHeader(http.StatusOK)
}

/**
  - PlanificationStop: Detiene el STS y LTS en caso de que la planificación se encuentre en ejecución. Si no, ignora la petición.
    El proceso que se encuentra en ejecución NO es desalojado. Una vez que salga de EXEC se pausa el manejo de su motivo de desalojo.
    El resto de procesos bloqueados van a pausar su transición a la cola de Ready
*/
func PlanificationStop(w http.ResponseWriter, r *http.Request) {
	globals.PlanningState = "STOPPED"
	globals.LTSPlanBinary <- true
	globals.STSPlanBinary <- true
	fmt.Println("Planification Stopped")
	w.WriteHeader(http.StatusOK)
}

type ProcessList_BRS struct {
	Pid   int    `json:"pid"`
	State string `json:"state"`
}

/**
 * ProcessList: Devuelve una lista de procesos con su PID y estado
*/
func ProcessList(w http.ResponseWriter, r *http.Request) {
	allProcesses := getProcessList()

	// Formateo los procesos para devolverlos
	respBody := make([]ProcessList_BRS, len(allProcesses))
	for i, process := range allProcesses {
		respBody[i] = ProcessList_BRS{Pid: int(process.PID), State: process.State}
	}

	response, err := json.Marshal(respBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

/**
  - getProcessList: Devuelve una lista de todos los procesos en el sistema (LTS, STS, Blocked, STS_Priority, CurrentJob)

  - @return []pcb.T_PCB: Lista de procesos
*/
func getProcessList() []pcb.T_PCB {
	var allProcesses []pcb.T_PCB
	allProcesses = append(allProcesses, globals.LTS...)
	allProcesses = append(allProcesses, globals.STS...)
	allProcesses = append(allProcesses, globals.STS_Priority...)
	allProcesses = append(allProcesses, globals.Blocked...)
	allProcesses = append(allProcesses, globals.Terminated...)
	if globals.CurrentJob.PID != 0 && pidIsNotOnList(globals.CurrentJob.PID, allProcesses){
		allProcesses = append(allProcesses, globals.CurrentJob)
	}
	return allProcesses
}

func pidIsNotOnList(pid uint32, list []pcb.T_PCB) bool {
	for _, process := range list {
		if process.PID == pid {
			return false
		}
	}
	return true
}

/**
  - PCB_Send: Envía un PCB al CPU y recibe la respuesta

  - @return error: Error en caso de que falle el envío
*/
func PCB_Send() error {
	jsonData, err := json.Marshal(globals.CurrentJob)
	if err != nil {
		return fmt.Errorf("failed to encode PCB: %v", err)
	}

	client := &http.Client{
		Timeout: 0,
	}

	// Send data
	url := fmt.Sprintf("http://%s:%d/dispatch", globals.Configkernel.IP_cpu, globals.Configkernel.Port_cpu)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("POST request failed. Failed to send PCB: %v", err)
	}

	// Wait for response
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	// Decode response and update value
	err = json.NewDecoder(resp.Body).Decode(&globals.CurrentJob) // ? Semaforo?
	if err != nil {
		return fmt.Errorf("failed to decode PCB response: %v", err)
	}

	globals.PcbReceived <- true

	return nil
}

/**
  - SearchByID: Busca un proceso en la lista de procesos en base a su PID

  - @param pid: PID del proceso a buscar
  - @param processList: Lista de procesos
  - @return *pcb.T_PCB: Proceso encontrado
*/
func SearchByID(pid uint32, processList []pcb.T_PCB) (*pcb.T_PCB, int) {
	if len(processList) == 0 {
		return nil, -1
	} else {
		for i, process := range processList {
			if process.PID == pid {
				return &process, i
			}
		}
	}
	return nil, -1
}

/**
  - DeleteByID: Remueve un proceso de la lista de procesos en base a su PID

  - @param pid: PID del proceso a remover
*/
func DeleteByID(pid uint32) error {
	pcbToDelete := RemoveByID(pid)

	if pcbToDelete.PID == 0 {
		return fmt.Errorf("process with PID %d not found", pid)
	} else {
		KillJob(pcbToDelete)
	}

	return nil
}

/**
  - RemoveByID: Remueve un proceso de la lista de procesos en base a su PID

  - @param pid: PID del proceso a remover
  - @return pcb.T_PCB: Proceso removido
*/
func RemoveByID(pid uint32) pcb.T_PCB {
	_, ltsIndex := SearchByID(pid, globals.LTS)
	_, stsIndex := SearchByID(pid, globals.STS)
	_, blockedIndex := SearchByID(pid, globals.Blocked)

	var removedPCB pcb.T_PCB

	if ltsIndex != -1 {
		globals.LTSMutex.Lock()
		defer globals.LTSMutex.Unlock()
		removedPCB = slice.RemoveAtIndex(&globals.LTS, ltsIndex)
		// Evita bloqueo de lista vacía
		if (len(globals.LTS) == 0) {
			globals.EmptiedList <- true
		}
	} else if stsIndex != -1 {
		globals.STSMutex.Lock()
		defer globals.STSMutex.Unlock()
		removedPCB = slice.RemoveAtIndex(&globals.STS, stsIndex)
		<- globals.MultiprogrammingCounter
		<- globals.STSCounter
	} else if blockedIndex != -1 {
		globals.BlockedMutex.Lock()
		defer globals.BlockedMutex.Unlock()
		removedPCB = slice.RemoveAtIndex(&globals.Blocked, blockedIndex)
	} else {
		return pcb.T_PCB{PID: 0} 
	}

	return removedPCB
}

func KillJob(pcb pcb.T_PCB) {
	globals.ChangeState(&pcb, "TERMINATED")
	if (resource.HasResources(pcb)) {
		advancedDeleting(pcb)
	}
	slice.Push(&globals.Terminated, pcb)
	RequestMemoryRelease(pcb.PID)
	fmt.Print("Se eliminó el proceso ", pcb.PID, " satisfactoriamente\n")
}

func advancedDeleting(pcb pcb.T_PCB) {
	for _ , res := range globals.Configkernel.Resources {
		if count, ok := pcb.Resources[res]; ok && count > 0 {
			pcb.Resources[res] = 0
			for range count {
				globals.Resource_instances[res]++
				resource.ReleaseJobIfBlocked(res)
			}
		}

		getIndex := func() int {
			for i, pcbResource := range globals.ResourceMap[res] {
				if pcbResource.PID == pcb.PID {
					return i
				}
			}
			return -1
		}

		index := getIndex()

		if index != -1 {
			globals.MapMutex.Lock()
			globals.ResourceMap[res] = append(globals.ResourceMap[res][:index], globals.ResourceMap[res][index+1:]...)
			globals.MapMutex.Unlock()
		}
	}
}

/**
  - GetPIDFromQueryPath: Convierte un PID en formato string a uint32

  - @param pidString: PID en formato string
  - @return uint32: PID extraído
*/
func GetPIDFromString(pidString string) (uint32, error) {
	pid64, error := strconv.ParseUint(pidString, 10, 32)
	return uint32(pid64), error
}

func RemoveFromBlocked(pid uint32) {
	for i, pcb := range globals.Blocked {
		if pcb.PID == pid {
			slice.RemoveAtIndex(&globals.Blocked, i)
		}
	}
}

type InterruptionRequest struct {
	InterruptionReason string `json:"InterruptionReason"`
	Pid                uint32 `json:"pid"`
	ExecutionNumber    int    `json:"execution_number"`
}

/**
 * SendInterrupt: Envia una interrupción a CPU

 * @param reason: Motivo de la interrupción
 * @param pid: PID del proceso a interrumpir
 * @param executionNumber: Número de ejecución del proceso
*/
func SendInterrupt(reason string, pid uint32, executionNumber int) {
	url := fmt.Sprintf("http://%s:%d/interrupt", globals.Configkernel.IP_cpu, globals.Configkernel.Port_cpu)

	bodyInt, err := json.Marshal(InterruptionRequest{
		InterruptionReason: reason,
		Pid:                pid,
		ExecutionNumber:    executionNumber,
	})
	if err != nil {
		return
	}

	enviarInterrupcion, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyInt))
	if err != nil {
		fmt.Printf("POST request failed (No se puede enviar interrupción): %v", err)
	}

	cliente := &http.Client{}
	enviarInterrupcion.Header.Set("Content-Type", "application/json")
	recibirRta, err := cliente.Do(enviarInterrupcion)
	if err != nil || recibirRta.StatusCode != http.StatusOK {
		fmt.Println("Error al interrupir proceso", err)
	}
}

/**
 * RequestMemoryDelay: Solicita el delay de memoria
*/
func RequestMemoryRelease(pid uint32) {
	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/finalizarProceso", globals.Configkernel.IP_memory, globals.Configkernel.Port_memory)

	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		fmt.Printf("Error al crear request para finalizar proceso: %v", err)
	}

	q := req.URL.Query()
	q.Add("pid", strconv.Itoa(int(pid)))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		fmt.Printf("Error al finalizar proceso en memoria: %v", err)
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		fmt.Printf("Error al finalizar proceso en memoria: %v", err)
	}
}

/**
 * GetPIDList: Devuelve una lista de PID de todos los procesos en el sistema

 * @param []pcb.T_PCB: Lista de procesos
 * @return []uint32: Lista de PID
*/
func GetPIDList([]pcb.T_PCB) []uint32 {
	var pidList []uint32
	for _, pcb := range getProcessList() {
		pidList = append(pidList, pcb.PID)
	}
	return pidList
}