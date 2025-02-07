package main

import (
	"fmt"
	"net/http"

	kernel_api "github.com/sisoputnfrba/tp-golang/kernel/API"
	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	resources "github.com/sisoputnfrba/tp-golang/kernel/resources"
	kernelutils "github.com/sisoputnfrba/tp-golang/kernel/utils"
	cfg "github.com/sisoputnfrba/tp-golang/utils/config"
	logger "github.com/sisoputnfrba/tp-golang/utils/log"
	"github.com/sisoputnfrba/tp-golang/utils/server-Functions"
)

func main() {
	logger.ConfigurarLogger("kernel.log")
	logger.LogfileCreate("kernel_debug.log")

	err := cfg.ConfigInit("config_kernel.json", &globals.Configkernel)
	if err != nil {
		fmt.Printf("Error al cargar la configuracion %v", err)
	}

	cfg.VEnvKernel(nil, &globals.Configkernel.Port)
	cfg.VEnvCpu(&globals.Configkernel.IP_cpu, &globals.Configkernel.Port_cpu)
	cfg.VEnvMemoria(&globals.Configkernel.IP_memory, &globals.Configkernel.Port_memory)

	fmt.Println("Configuracion KERNEL cargada")

	globals.MultiprogrammingCounter = make (chan int, globals.Configkernel.Multiprogramming)
	globals.STSCounter = make (chan int, globals.Configkernel.Multiprogramming)
	resources.InitResourceMap()

	globals.EmptiedList <- false
	globals.LTSPlanBinary <- false
	globals.STSPlanBinary <- false
	globals.PlanningState = "STOPPED"

	go ServerStart(globals.Configkernel.Port)

	// * Planificación
	go kernelutils.LTS_Plan()
	go kernelutils.STS_Plan()

	select {}
}

func ServerStart(port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("/paquetes", 				server.RecibirPaquetes)
	mux.HandleFunc("/mensaje", 					server.RecibirMensaje)
	// Procesos
	mux.HandleFunc("GET /process",				kernel_api.ProcessList)
	mux.HandleFunc("PUT /process",				kernel_api.ProcessInit)
	mux.HandleFunc("GET /process/{pid}", 		kernel_api.ProcessState)
	mux.HandleFunc("DELETE /process/{pid}",		kernel_api.ProcessDelete)
	// Planificación
	mux.HandleFunc("PUT /plani", 				kernel_api.PlanificationStart)
	mux.HandleFunc("DELETE /plani",				kernel_api.PlanificationStop)
	// I/O
	mux.HandleFunc("POST /io-handshake", 		kernel_api.GetIOInterface)
	mux.HandleFunc("POST /io-interface", 		kernel_api.ExisteInterfaz)
	mux.HandleFunc("POST /iodata-gensleep",		kernel_api.RecvData_gensleep)
	mux.HandleFunc("POST /iodata-stdin", 		kernel_api.RecvData_stdin)
	mux.HandleFunc("POST /iodata-stdout", 		kernel_api.RecvData_stdout)
	mux.HandleFunc("POST /iodata-dialfs", 		kernel_api.RecvData_dialfs)
	mux.HandleFunc("POST /io-return-pcb", 		kernel_api.RecvPCB_IO)
	// Recursos
	mux.HandleFunc("GET /resource-info", 		resources.GETResourcesInstances)
	mux.HandleFunc("GET /resourceblocked", 		resources.GETResourceBlockedJobs)

	fmt.Printf("Server listening on port %d\n", port)
	err := http.ListenAndServe(":"+fmt.Sprintf("%v", port), mux)
	if err != nil {
		panic(err)
	}
}