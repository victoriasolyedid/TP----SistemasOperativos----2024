package globals

import (
	"log"
	"sync"

	"github.com/sisoputnfrba/tp-golang/utils/device"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
)

// Global variables
var (
	NextPID 					uint32 = 0
	LTS 						[]pcb.T_PCB
	STS 						[]pcb.T_PCB
	Blocked 					[]pcb.T_PCB
	STS_Priority 				[]pcb.T_PCB
	Terminated 					[]pcb.T_PCB
	Interfaces 					[]device.T_IOInterface
	ResourceMap					map[string][]pcb.T_PCB
	Resource_instances  		map[string]int
	PlanningState				string
)

// Global semaphores
var (
	// * Mutex
		PidMutex 				sync.Mutex
		ProcessesMutex 			sync.Mutex
		STSMutex 				sync.Mutex
		LTSMutex 				sync.Mutex
		BlockedMutex			sync.Mutex
		MapMutex 				sync.Mutex
		EnganiaPichangaMutex	sync.Mutex
	// * Binarios
		LTSPlanBinary  			= make (chan bool, 1)
		STSPlanBinary  			= make (chan bool, 1)
		JobExecBinary			= make (chan bool, 1)
		PcbReceived				= make (chan bool, 1)
		AvailablePcb			= make (chan bool, 1)
		EmptiedList				= make (chan bool, 1)
	// * Contadores
		// Chequea si hay procesos en la cola de listos, lo usamos en EvictionManagement y en ProcessInit
		MultiprogrammingCounter chan int
		STSCounter 				chan int
)

var CurrentJob pcb.T_PCB

type T_ConfigKernel struct {
	Port 						int 		`json:"port"`
	IP_memory 					string 		`json:"ip_memory"`
	Port_memory 				int 		`json:"port_memory"`
	IP_cpu 						string 		`json:"ip_cpu"`
	Port_cpu 					int 		`json:"port_cpu"`
	Planning_algorithm 			string 		`json:"planning_algorithm"`
	Quantum 					uint32 		`json:"quantum"`
	Resources 					[]string 	`json:"resources"`
	Resource_instances 			[]int 		`json:"resource_instances"`
	Multiprogramming 			int 		`json:"multiprogramming"`
}

var Configkernel *T_ConfigKernel

func ChangeState(pcb *pcb.T_PCB, newState string) {
	ProcessesMutex.Lock()
	defer ProcessesMutex.Unlock()
	
	prevState := pcb.State
	pcb.State = newState
	log.Printf("PID: %d - Estado anterior: %s - Estado actual: %s \n", pcb.PID, prevState, pcb.State)
}
		
var BlockedJob_by_IO pcb.T_PCB

type DireccionTamanio struct {
	DireccionFisica 	int 
	Tamanio         	int 
}