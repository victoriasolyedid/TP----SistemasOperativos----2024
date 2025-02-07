package resource

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
	"github.com/sisoputnfrba/tp-golang/utils/slice"
)

/**
 * InitResourceMap: Inicializa el mapa de recursos y la cantidad de instancias de cada recurso
 */
func InitResourceMap() {
	globals.ResourceMap = make(map[string][]pcb.T_PCB)
	globals.Resource_instances = make(map[string]int)

	for i, resource := range globals.Configkernel.Resources {
		globals.ResourceMap[resource] = []pcb.T_PCB{}
		globals.Resource_instances[resource] = globals.Configkernel.Resource_instances[i]
	}
}

/**
 * QueueProcess: Encola un proceso en la cola de bloqueo de un recurso

 * @param resource: recurso al que se quiere acceder
 * @param pcb: proceso a encolar
*/
func QueueProcess(resource string, pcb pcb.T_PCB) {
	globals.ResourceMap[resource] = append(globals.ResourceMap[resource], pcb)
	slice.Push(&globals.Blocked, pcb)
}

/**
 * DequeueProcess: Desencola un proceso de la cola de bloqueo de un recurso

 * @param resource: recurso al que se quiere acceder
 * @return pcb: proceso desencolado
*/
func DequeueProcess(resource string) pcb.T_PCB {
	pcb := globals.ResourceMap[resource][0]
	globals.ResourceMap[resource] = globals.ResourceMap[resource][1:]

	RemoveFromBlocked(uint32(pcb.PID))
	return pcb
}

/**
 * RemoveFromBlocked: Remueve un proceso de la cola de bloqueo

 * @param pcb: proceso a remover
*/
func RemoveFromBlocked(pid uint32) {
	for i, pcb := range globals.Blocked {
		if pcb.PID == pid {
			slice.RemoveAtIndex(&globals.Blocked, i)
		}
	}
}

/**
 * RequestConsumption: Solicita la consumisión una instancia de un recurso

 * @param resource: recurso a consumir
*/
func RequestConsumption(resource string) {
	globals.MapMutex.Lock()
	defer globals.MapMutex.Unlock()
	if IsAvailable(resource) {
		globals.ChangeState(&globals.CurrentJob, "READY")
		globals.Resource_instances[resource]--
		globals.CurrentJob.Resources[resource]++
		fmt.Print("Se consumio una instancia del recurso: ", resource, "\n")
		globals.CurrentJob.RequestedResource = ""
		slice.Push(&globals.STS, globals.CurrentJob)
		globals.STSCounter <- 1
	} else {
		fmt.Print("No hay instancias del recurso solicitado\n")
		globals.ChangeState(&globals.CurrentJob, "BLOCKED")
		globals.CurrentJob.PC--	// Se decrementa el PC para que no avance en la próxima ejecución
		log.Print("PID: ", globals.CurrentJob.PID, " - Bloqueado por: ", resource, "\n")
		fmt.Print("Entra el proceso PID: ", globals.CurrentJob.PID, " a la cola de bloqueo del recurso ", resource,  "\n")
		QueueProcess(resource, globals.CurrentJob)
	}
}

/**
 * ReleaseConsumption: Solicita la liberación de una instancia de un recurso

 * @param resource: recurso a liberar
*/
func ReleaseConsumption(resource string) {
	globals.MapMutex.Lock()
	defer globals.MapMutex.Unlock()

	if globals.CurrentJob.Resources[resource] == 0 {
		fmt.Print("El proceso PID: ", globals.CurrentJob.PID, " no tiene instancias del recurso ", resource, " para liberar\n")
		return
	}

	globals.CurrentJob.Resources[resource]--
	globals.Resource_instances[resource]++
	fmt.Print("Se libero una instancia del recurso: ", resource, "\n")
	slice.InsertAtIndex(&globals.STS, 0, globals.CurrentJob)
	ReleaseJobIfBlocked(resource)
	globals.STSCounter <- 1
}

/**
 * Exists: Consulta si existe un recurso

 * @param resource: recurso a consultar
 * @return bool: true si existe, false en caso contrario
*/
func Exists(resource string) bool {
	globals.MapMutex.Lock()
	defer globals.MapMutex.Unlock()

	_, ok := globals.Resource_instances[resource]
	return ok
}

/**
 * IsAvailable: Consulta si hay instancias disponibles de un recurso

 * @param resource: recurso a consultar
 * @return bool: true si hay instancias disponibles, false en caso contrario
*/
func IsAvailable(resource string) bool {
	return globals.Resource_instances[resource] > 0
}

/**
 * ReleaseJobIfBlocked: Libera un proceso bloqueado por un recurso

 * @param resource: recurso del que se quiere liberar un proceso
*/
func ReleaseJobIfBlocked(resource string) {
	if len(globals.ResourceMap[resource]) > 0 {
		pcb := DequeueProcess(resource)
		globals.ChangeState(&pcb, "READY")
		globals.STS = append(globals.STS, pcb)
		fmt.Print("Se desbloqueo el proceso PID: ", pcb.PID, " del recurso ", resource, "\n")
		globals.STSCounter <- 1
	}
}

/**
 * ReleaseAllResources: Libera todos los recursos de un proceso

 * @param pcb: proceso al que se le quieren liberar los recursos
 * @return pcb: proceso con los recursos liberados
*/
func ReleaseAllResources(pcb pcb.T_PCB) pcb.T_PCB {
	for resource, instances := range globals.CurrentJob.Resources {
		for i := 0; i < instances; i++ {
			ReleaseConsumption(resource)
		}
	}
	
	return pcb
}

/**
 * HasResources: Consulta si un proceso tiene recursos

 * @param pcb: proceso a consultar
 * @return bool: true si tiene recursos, false en caso contrario
*/
func HasResources(pcb pcb.T_PCB) bool {
	for _, instances := range pcb.Resources {
		if instances > 0 {
			return true
		}
	}
	return false
}


// --------------------- API ------------------------

/**
 * GETResourcesInstances: Consulta la cantidad de instancias de cada recurso

 * @param w: response writer
 * @param r: request
*/
func GETResourcesInstances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globals.Resource_instances)
}

/**
 * GETResourceBlockedJobs: Consulta los procesos bloqueados por recurso

 * @param w: response writer
 * @param r: request
*/
func GETResourceBlockedJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globals.ResourceMap)
}