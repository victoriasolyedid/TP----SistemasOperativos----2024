package cpu_api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/cicloInstruccion"
	"github.com/sisoputnfrba/tp-golang/cpu/globals"
	"github.com/sisoputnfrba/tp-golang/utils/generics"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
)

/**
 * PCB_recv: Recibe un PCB, lo "procesa" y lo devuelve
 * Cumple con la funcionalidad principal de CPU.
	* Procesar = Fetch -> Decode -> Execute
*/
func PCB_recv(w http.ResponseWriter, r *http.Request) {
	var received_pcb pcb.T_PCB

	// Decode PCB
	err := json.NewDecoder(r.Body).Decode(&received_pcb)
	if err != nil {
		http.Error(w, "Failed to decode PCB", http.StatusBadRequest)
		return
	}

	globals.CurrentJob = &received_pcb

	for {
		globals.EvictionMutex.Lock()
		if pcb.EvictionFlag { 
			globals.EvictionMutex.Unlock() 
			break }
		globals.EvictionMutex.Unlock()
		
		if (globals.MemDelay > int(globals.CurrentJob.Quantum)) {
			globals.CurrentJob.EvictionReason = "TIMEOUT"
			pcb.EvictionFlag = true
		}
		cicloInstruccion.DecodeAndExecute(globals.CurrentJob)
		
		fmt.Println("Los registros de la cpu son", globals.CurrentJob.CPU_reg)

	}

	fmt.Println("CPU - El motivo de la interrupción es: ", globals.CurrentJob.EvictionReason)
	//fmt.Println("ABER MOSTRAMELON: ", pcb.EvictionFlag) // * Se recordará su contribución a la ciencia
	pcb.EvictionFlag = false

	jsonResp, err := json.Marshal(globals.CurrentJob)
	if err != nil {
		http.Error((w), "Failed to encode PCB response", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

type InterruptionRequest struct {
	InterruptionReason string `json:"InterruptionReason"`
	Pid                uint32 `json:"pid"`
	ExecutionNumber    int    `json:"execution_number"`
}

/**
 * HandleInterruption: Maneja las interrupciones de CPU
 */
func HandleInterruption(w http.ResponseWriter, r *http.Request) {
	var request InterruptionRequest

	// Decode json payload
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	fmt.Println("PID Requerido: ", request.Pid)
	fmt.Println("PID Actual: ", globals.CurrentJob.PID)
	
	if _, ok := globals.EvictionReasons[globals.CurrentJob.EvictionReason]; !ok && request.Pid == globals.CurrentJob.PID && (globals.CurrentJob.Executions == request.ExecutionNumber || request.ExecutionNumber == -1) {
		fmt.Println("Se acepta interrumpir PID: ", request.Pid)
		fmt.Printf("Motivo de interrupción: %s\n", request.InterruptionReason)

		globals.EvictionMutex.Lock()
		pcb.EvictionFlag = true
		globals.EvictionMutex.Unlock()

		switch request.InterruptionReason {
		case "QUANTUM":
			globals.CurrentJob.EvictionReason = "TIMEOUT"

		case "DELETE":
			globals.CurrentJob.EvictionReason = "INTERRUPTED_BY_USER"
		}
	}

	w.WriteHeader(http.StatusOK)
}

func RequestMemoryDelay() {
	url := fmt.Sprintf("http://%s:%d/delay", globals.Configcpu.IP_memory, globals.Configcpu.Port_memory)

	var delayStruct struct {
		Delay int
	}

	generics.DoRequest("GET", url, nil, &delayStruct)

	globals.MemDelay = delayStruct.Delay
}
