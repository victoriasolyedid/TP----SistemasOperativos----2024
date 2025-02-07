package globals

import (
	"github.com/sisoputnfrba/tp-golang/utils/device"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
)

var (
	CurrentBitMap []byte
	Blocks []byte
	Fcbs map[string]Metadata
)

type T_ConfigIO struct {
	Ip                 		string `json:"ip"`
	Port               		int    `json:"port"`
	Type               		string `json:"type"`
	Unit_work_time     		int    `json:"unit_work_time"`
	Ip_kernel          		string `json:"ip_kernel"`
	Port_kernel        		int    `json:"port_kernel"`
	Ip_memory          		string `json:"ip_memory"`
	Port_memory        		int    `json:"port_memory"`
	Dialfs_path        		string `json:"dialfs_path"`
	Dialfs_block_size  		int    `json:"dialfs_block_size"`
	Dialfs_block_count 		int    `json:"dialfs_block_count"`
	Dialfs_compaction_delay int    `json:"dialfs_compaction_delay"`
}

// ----------------- Body types -----------------

type GenSleep struct {
	Pcb         			pcb.T_PCB
	Inter       			device.T_IOInterface
	TimeToSleep 			int
}

type StdinRead struct {
	Pcb 					pcb.T_PCB
	Inter 					device.T_IOInterface
	DireccionesFisicas 		[]DireccionTamanio
}

type StdoutWrite struct {
	Pcb 					pcb.T_PCB
	Inter 					device.T_IOInterface
	DireccionesFisicas 		[]DireccionTamanio
}

type DialFSRequest struct {
	Pcb 					pcb.T_PCB
	Inter 					device.T_IOInterface
	NombreArchivo 			string
	Tamanio 				int
	Puntero					int
	Direccion				[]DireccionTamanio
	Operacion				string
}

var ConfigIO 				T_ConfigIO
var Generic_QueueChannel 	chan GenSleep
var Stdin_QueueChannel 		chan StdinRead
var Stdout_QueueChannel 	chan StdoutWrite
var DialFS_QueueChannel 	chan DialFSRequest

type DireccionTamanio struct {
	DireccionFisica 		int
	Tamanio         		int
}

type Metadata struct {
	InitialBlock 			int 	`json:"initial_block"`
	Size 					int 	`json:"size"`
}