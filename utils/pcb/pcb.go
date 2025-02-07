package pcb

// Estructura PCB que comparten tanto el kernel como el CPU
type T_PCB struct {
	PID 				uint32 						`json:"pid"`
	PC 					uint32 						`json:"pc"`
	Quantum 			uint32 						`json:"quantum"`
	CPU_reg 			map[string]interface{} 		`json:"cpu_reg"`	
	State 				string 						`json:"state"`
	EvictionReason 		string  					`json:"eviction_reason"`
	Resources 			map[string]int				`json:"resources"`
	RequestedResource 	string 						`json:"requested_resource"`
	Executions 			int 						`json:"executions"`
}

func TipoReg(reg string) string {
	if reg == "AX" || reg == "BX" || reg == "CX" || reg == "DX" {
		return "uint8"
	} else {
		return "uint32"
	}
}

var EvictionFlag bool = false