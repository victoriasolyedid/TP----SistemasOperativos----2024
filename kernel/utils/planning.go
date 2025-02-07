package kernelutils

import (
	"fmt"
	"log"
	"time"

	kernel_api "github.com/sisoputnfrba/tp-golang/kernel/API"
	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	resource "github.com/sisoputnfrba/tp-golang/kernel/resources"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
	"github.com/sisoputnfrba/tp-golang/utils/slice"
)

func LTS_Plan() {
	for {

		if globals.PlanningState == "STOPPED" {
			globals.LTSPlanBinary <- true
			<- globals.LTSPlanBinary
			continue
		}

		if len(globals.LTS) == 0 {
			globals.EmptiedList <- true
			continue
		}

		globals.LTSMutex.Lock()
		auxJob := slice.Shift(&globals.LTS)
		globals.LTSMutex.Unlock()
		if auxJob.PID != 0 {
			globals.MultiprogrammingCounter <- int(auxJob.PID)
			globals.ChangeState(&auxJob, "READY")
			slice.Push(&globals.STS, auxJob)
			log.Printf("Cola Ready STS: %v", kernel_api.GetPIDList(globals.STS))
			globals.STSCounter <- int(auxJob.PID)
		}
	}
}

func STS_Plan() {
	switch globals.Configkernel.Planning_algorithm {
	case "FIFO":
		fmt.Println("FIFO algorithm")
		for {
			if globals.PlanningState == "STOPPED" {
				globals.STSPlanBinary <- true
				<- globals.STSPlanBinary
				continue
			}

			<-globals.STSCounter
			FIFO_Plan()
		}

	case "RR":
		fmt.Println("ROUND ROBIN algorithm")
		for {
			if globals.PlanningState == "STOPPED" {
				globals.STSPlanBinary <- true
				<- globals.STSPlanBinary
				continue
			}
			<-globals.STSCounter
			RR_Plan()
		}

	case "VRR":
		fmt.Println("VIRTUAL ROUND ROBIN algorithm")
		for {
			if globals.PlanningState == "STOPPED" {
				globals.STSPlanBinary <- true
				<- globals.STSPlanBinary
				continue
			}

			<-globals.STSCounter
			VRR_Plan()
		}

	default:
		fmt.Println("Not a planning algorithm")
	}
}

type T_Quantum struct {
	TimeExpired chan bool
}

/**
  - FIFO_Plan
*/
func FIFO_Plan() {
	globals.CurrentJob = slice.Shift(&globals.STS)

	globals.ChangeState(&globals.CurrentJob, "EXEC")
	globals.CurrentJob.Executions++

	kernel_api.PCB_Send()

	<-globals.PcbReceived

	EvictionManagement()
}

func RR_Plan() {
	globals.EnganiaPichangaMutex.Lock()
	
	globals.CurrentJob = slice.Shift(&globals.STS)
	globals.ChangeState(&globals.CurrentJob, "EXEC")
	globals.CurrentJob.Executions++
	globals.EnganiaPichangaMutex.Unlock()

	go startTimer(globals.CurrentJob.Quantum)
	kernel_api.PCB_Send()                                            

	<-globals.PcbReceived

	EvictionManagement()
}

func VRR_Plan() {
    globals.EnganiaPichangaMutex.Lock()

    if len(globals.STS_Priority) > 0 {
        globals.CurrentJob = slice.Shift(&globals.STS_Priority)
    } else {
        globals.CurrentJob = slice.Shift(&globals.STS)
    }

    globals.ChangeState(&globals.CurrentJob, "EXEC")
	globals.CurrentJob.Executions++
    globals.EnganiaPichangaMutex.Unlock()

    timeBefore := time.Now()
    go startTimer(globals.CurrentJob.Quantum)

    kernel_api.PCB_Send()

    <-globals.PcbReceived

    // Calcular el tiempo que tomó la ejecución
    timeAfter := time.Now()
    diffTime := uint32(time.Duration(timeAfter.Sub(timeBefore)).Milliseconds())

    if diffTime < globals.CurrentJob.Quantum {
        globals.CurrentJob.Quantum = globals.CurrentJob.Quantum - diffTime
		fmt.Print("Quantum restante: ", globals.CurrentJob.Quantum)
    } else {
        globals.CurrentJob.Quantum = globals.Configkernel.Quantum
    }

    EvictionManagement()
}

func startTimer(quantum uint32) {
	quantumTime := time.Duration(quantum) * time.Millisecond
	fmt.Println("Quantum time: ", quantumTime)
	auxPcb := globals.CurrentJob

	timeBefore := time.Now()
	
	time.Sleep(quantumTime)
	
	timeAfter := time.Now()
    diffTime := uint32(time.Duration(timeAfter.Sub(timeBefore)).Milliseconds())
	fmt.Println("START TIMER - Difftime: ", diffTime)	

	fmt.Println("Salió de mimir el PID", auxPcb.PID)
	quantumInterrupt(auxPcb)
}

func quantumInterrupt(pcb pcb.T_PCB) {
	fmt.Printf("Mando interrupción por quantum al PID %d\n", pcb.PID)
	fmt.Println("La eviction reason es:", pcb.EvictionReason)

	kernel_api.SendInterrupt("QUANTUM", pcb.PID, pcb.Executions)
}

/**
  - EvictionManagement
*/
func EvictionManagement() {
	evictionReason := globals.CurrentJob.EvictionReason
	globals.CurrentJob.EvictionReason = ""

	switch evictionReason {
	case "BLOCKED_IO_GEN":
		globals.EnganiaPichangaMutex.Lock()
		globals.ChangeState(&globals.CurrentJob, "BLOCKED")
		
		pcbAux := globals.CurrentJob
		slice.Push(&globals.Blocked, globals.CurrentJob)
		log.Printf("PID: %d - Bloqueado por I/O GENERICA\n", globals.CurrentJob.PID)
		go func() {
			kernel_api.SolicitarGenSleep(pcbAux)
		}()

	case "BLOCKED_IO_STDIN":
		globals.EnganiaPichangaMutex.Lock()
		globals.ChangeState(&globals.CurrentJob, "BLOCKED")
		
		pcbAux := globals.CurrentJob
		slice.Push(&globals.Blocked, globals.CurrentJob)
		log.Printf("PID: %d - Bloqueado por I/O STDIN\n", globals.CurrentJob.PID)
		go func() {
			kernel_api.SolicitarStdinRead(pcbAux)
		}()

	case "BLOCKED_IO_STDOUT":
		globals.EnganiaPichangaMutex.Lock()
		globals.ChangeState(&globals.CurrentJob, "BLOCKED")
		
		pcbAux := globals.CurrentJob
		slice.Push(&globals.Blocked, globals.CurrentJob)
		log.Printf("PID: %d - Bloqueado por I/O STDOUT\n", globals.CurrentJob.PID)
		go func() {
			kernel_api.SolicitarStdoutWrite(pcbAux)
		}()

	case "BLOCKED_IO_DIALFS":
		globals.EnganiaPichangaMutex.Lock()
		globals.ChangeState(&globals.CurrentJob, "BLOCKED")

		pcbAux := globals.CurrentJob
		slice.Push(&globals.Blocked, globals.CurrentJob)
		log.Printf("PID: %d - Bloqueado por I/O DIALFS\n", globals.CurrentJob.PID)
		go func() {
			kernel_api.SolicitarDialFS(pcbAux)
		}()

	case "TIMEOUT":
		globals.ChangeState(&globals.CurrentJob, "READY")
		globals.STS = append(globals.STS, globals.CurrentJob)
		log.Printf("PID: %d - Desalojado por fin de quantum\n", globals.CurrentJob.PID)
		globals.STSCounter <- int(globals.CurrentJob.PID)

	case "EXIT":
		globals.ChangeState(&globals.CurrentJob, "TERMINATED")
		kernel_api.KillJob(globals.CurrentJob)
		<-globals.MultiprogrammingCounter
		log.Printf("Finaliza el proceso %d - Motivo: %s\n", globals.CurrentJob.PID, evictionReason)

	case "WAIT":
		if resource.Exists(globals.CurrentJob.RequestedResource) {
			resource.RequestConsumption(globals.CurrentJob.RequestedResource)

		} else {
			fmt.Print("El recurso no existe\n")
			globals.CurrentJob.EvictionReason = "EXIT"
			EvictionManagement()
		}

	case "SIGNAL":
		if resource.Exists(globals.CurrentJob.RequestedResource) {
			resource.ReleaseConsumption(globals.CurrentJob.RequestedResource)

		} else {
			fmt.Print("El recurso no existe\n")
			globals.CurrentJob.EvictionReason = "EXIT"
			EvictionManagement()
		}

	case "OUT_OF_MEMORY":
		globals.ChangeState(&globals.CurrentJob, "TERMINATED")
		kernel_api.KillJob(globals.CurrentJob)
		<-globals.MultiprogrammingCounter
		log.Printf("Finaliza el proceso %d - Motivo: %s\n", globals.CurrentJob.PID, evictionReason)

	case "INTERRUPTED_BY_USER":
		globals.ChangeState(&globals.CurrentJob, "TERMINATED")
		kernel_api.KillJob(globals.CurrentJob)
		<-globals.MultiprogrammingCounter
		log.Printf("Finaliza el proceso %d - Motivo: %s\n", globals.CurrentJob.PID, evictionReason)

	default:
		fmt.Printf("'%s' no es una razón de desalojo válida", evictionReason)
	}
}