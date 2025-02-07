package main

import (
	"fmt"
	"log"
	"net/http"

	cpu_api "github.com/sisoputnfrba/tp-golang/cpu/API"
	"github.com/sisoputnfrba/tp-golang/cpu/globals"
	cfg "github.com/sisoputnfrba/tp-golang/utils/config"
	logger "github.com/sisoputnfrba/tp-golang/utils/log"
	server "github.com/sisoputnfrba/tp-golang/utils/server-Functions"
)

func main() {
	// Iniciar loggers
	logger.ConfigurarLogger("cpu.log")
	logger.LogfileCreate("cpu_debug.log")

	// *** CONFIGURACION ***
	err := cfg.ConfigInit("config-cpu.json", &globals.Configcpu)
	if err != nil {
		log.Fatalf("Error al cargar la configuracion %v", err)
	}

	cfg.VEnvCpu(nil, &globals.Configcpu.Port)
	cfg.VEnvKernel(&globals.Configcpu.IP_kernel, &globals.Configcpu.Port_kernel)
	cfg.VEnvMemoria(&globals.Configcpu.IP_memory, &globals.Configcpu.Port_memory)
	
	fmt.Println("Configuracion CPU cargada")

	// Handlers
	cpuRoutes := RegisteredModuleRoutes()

	// *** SERVIDOR ***
	go server.ServerStart(globals.Configcpu.Port, cpuRoutes)
	cpu_api.RequestMemoryDelay()

	// *** CLIENTE ***
	
	select {}
}

func RegisteredModuleRoutes() http.Handler {
	moduleHandler := &server.ModuleHandler{
		RouteHandlers: map[string]http.HandlerFunc{
			"POST /dispatch": 	cpu_api.PCB_recv,
			"POST /interrupt": 	cpu_api.HandleInterruption,
		},
	}
	return moduleHandler
}
