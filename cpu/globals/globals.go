package globals

import (
	"encoding/binary"
	"log"
	"strconv"
	"sync"

	"github.com/sisoputnfrba/tp-golang/utils/device"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
)

var Configcpu *T_CPU
var MemDelay int

var EvictionReasons = map[string]struct{}{
		"EXIT":          		{},
		"BLOCKED_IO_GEN": 		{},
		"BLOCKED_IO_STDIN":		{},
		"BLOCKED_IO_STDOUT":	{},
		"BLOCKED_IO_DIALFS":    {},
		"OUT_OF_MEMORY": 		{},
		"WAIT":		 			{},
		"SIGNAL":		 		{},
	}

type T_CPU struct {
	Port               int    `json:"port"`
	IP_memory          string `json:"ip_memory"`
	Port_memory        int    `json:"port_memory"`
	IP_kernel          string `json:"ip_kernel"`
	Port_kernel        int    `json:"port_kernel"`
	Number_felling_tlb int    `json:"number_felling_tlb"`
	Algorithm_tlb      string `json:"algorithm_tlb"`
}

var CurrentJob *pcb.T_PCB

// Global semaphores
var (
	// * Mutex
	EvictionMutex  sync.Mutex
	OperationMutex sync.Mutex
	PCBMutex       sync.Mutex
	// * Binario
	PlanBinary = make(chan bool, 1)
	// * Contadores
	MultiprogrammingCounter = make(chan int, 10)
)

type Frame int

type InterfaceController struct {
	IoInterf   device.T_IOInterface
	Controller chan bool
}

type DireccionTamanio struct {
	DireccionFisica int
	Tamanio         int
}

func PasarAInt(cadena string) int {
	num, err := strconv.Atoi(cadena)
	if err != nil {
		log.Println("Error: ", err)
	}
	return num
}

func BytesToInt(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}
