package cicloInstruccion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"

	mmu "github.com/sisoputnfrba/tp-golang/cpu/mmu"

	solicitudesmemoria "github.com/sisoputnfrba/tp-golang/cpu/solicitudesMemoria"
	"github.com/sisoputnfrba/tp-golang/utils/pcb"
)

/**
 * Delimitador: Función que separa la instrucción en sus partes
 * @param instActual: Instrucción a separar
 * @return instruccionDecodificada: Instrucción separada
**/

func Delimitador(instActual string) []string {
	delimitador := " "
	i := 0

	instruccionDecodificadaConComillas := strings.Split(instActual, delimitador)
	instruccionDecodificada := instruccionDecodificadaConComillas

	largoInstruccion := len(instruccionDecodificadaConComillas)
	for i < largoInstruccion {
		instruccionDecodificada[i] = strings.Trim(instruccionDecodificadaConComillas[i], `"`)
		i++
	}

	return instruccionDecodificada
}

func Fetch(currentPCB *pcb.T_PCB) string {
	// CPU pasa a memoria el PID y el PC, y memoria le devuelve la instrucción
	// (después de identificar en el diccionario la key: PID,
	// va a buscar en la lista de instrucciones de ese proceso, la instrucción en la posición
	// pc y nos va a devolver esa instrucción)
	// GET /instrucciones
	globals.PCBMutex.Lock()
	pid := currentPCB.PID
	pc := currentPCB.PC
	globals.PCBMutex.Unlock()

	cliente := &http.Client{}
	url := fmt.Sprintf("http://%s:%d/instrucciones", globals.Configcpu.IP_memory, globals.Configcpu.Port_memory)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Print("Error al crear el request")
	}
	q := req.URL.Query()
	q.Add("pid", strconv.Itoa(int(pid)))
	q.Add("pc", strconv.Itoa(int(pc)))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		fmt.Print("Error al hacer el request")
	}

	if respuesta.StatusCode != http.StatusOK {
		fmt.Print("Error en el estado de la respuesta")
	}

	instruccion, err := io.ReadAll(respuesta.Body)
	if err != nil {
		fmt.Print("Error al leer el cuerpo de la respuesta")
	}

	instruccion1 := string(instruccion)

	log.Printf("PID: %d - FETCH - Program Counter: %d", pid, pc)

	return instruccion1
}

func DecodeAndExecute(currentPCB *pcb.T_PCB) {
	instActual := Fetch(currentPCB)
	instruccionDecodificada := Delimitador(instActual)

	if instruccionDecodificada[0] == "EXIT" {
		currentPCB.EvictionReason = "EXIT"
		pcb.EvictionFlag = true

		log.Printf("PID: %d - Ejecutando: %s", currentPCB.PID, instruccionDecodificada[0])
	} else {
		log.Printf("PID: %d - Ejecutando: %s - %s", currentPCB.PID, instruccionDecodificada[0], instruccionDecodificada[1:])
	}

	currentPCB.PC++
	currentPCB.CPU_reg["PC"] = uint32(currentPCB.PC)

	switch instruccionDecodificada[0] {
	case "IO_FS_CREATE":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "DIALFS")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO Genéricas")
			currentPCB.EvictionReason = "EXIT"
		} else {
			nombre_archivo := instruccionDecodificada[2]
			if cond {

				var fsCreateBody = struct {
					InterfaceName string
					FileName      string
					Operation     string
				}{
					InterfaceName: instruccionDecodificada[1],
					FileName:      nombre_archivo,
					Operation:     "CREATE",
				}

				SendIOData(fsCreateBody, "iodata-dialfs")
				currentPCB.EvictionReason = "BLOCKED_IO_DIALFS"
			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true

	case "IO_FS_DELETE":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "DIALFS")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO Genéricas")
			currentPCB.EvictionReason = "EXIT"
		} else {
			nombre_archivo := instruccionDecodificada[2]
			if cond {

				var fsCreateBody = struct {
					InterfaceName string
					FileName      string
					Operation     string
				}{
					InterfaceName: instruccionDecodificada[1],
					FileName:      nombre_archivo,
					Operation:     "DELETE",
				}

				SendIOData(fsCreateBody, "iodata-dialfs")
				currentPCB.EvictionReason = "BLOCKED_IO_DIALFS"
			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true

		/* (Interfaz, Nombre Archivo, Registro Tamaño): Esta instrucción
		solicita al Kernel que mediante la interfaz seleccionada, se modifique
		el tamaño del archivo en el FS montado en dicha interfaz, actualizando al
		valor que se encuentra en el registro indicado por Registro Tamaño.
		*/
	case "IO_FS_TRUNCATE":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "DIALFS")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO Genéricas")
			currentPCB.EvictionReason = "EXIT"
		} else {
			nombre_archivo := instruccionDecodificada[2]
			tamanio_archivo := currentPCB.CPU_reg[instruccionDecodificada[3]]
			var tamanioEnInt int

			tipoActualReg2 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[3]]).String()

			if tipoActualReg2 == "uint32" {
				tamanioEnInt = int(Convertir[uint32](tipoActualReg2, tamanio_archivo))
			} else {
				tamanioEnInt = int(Convertir[uint8](tipoActualReg2, tamanio_archivo))
			}

			if cond {

				var fsCreateBody = struct {
					InterfaceName string
					FileName      string
					Size          int
					Operation     string
				}{
					InterfaceName: instruccionDecodificada[1],
					FileName:      nombre_archivo,
					Size:          tamanioEnInt,
					Operation:     "TRUNCATE",
				}

				SendIOData(fsCreateBody, "iodata-dialfs")
				currentPCB.EvictionReason = "BLOCKED_IO_DIALFS"

			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true
		/* IO_FS_WRITE (Interfaz, Nombre Archivo, Registro Dirección, Registro Tamaño, Registro Puntero Archivo):
		   Esta instrucción solicita al Kernel que mediante la interfaz seleccionada, se lea desde Memoria la cantidad
		    de bytes indicadas por el Registro Tamaño a partir de la dirección lógica que se encuentra en el Registro
		    Dirección y se escriban en el archivo a partir del valor del Registro Puntero Archivo.
		*/
	case "IO_FS_WRITE":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "DIALFS")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO Genéricas")
			currentPCB.EvictionReason = "EXIT"
		} else {
			nombre_archivo := instruccionDecodificada[2]
			direccion := currentPCB.CPU_reg[instruccionDecodificada[3]]
			var direccionEnInt int

			tipoActualRegDireccion := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[3]]).String()

			if tipoActualRegDireccion == "uint32" {
				direccionEnInt = int(Convertir[uint32](tipoActualRegDireccion, direccion))
			} else {
				direccionEnInt = int(Convertir[uint8](tipoActualRegDireccion, direccion))
			}

			tamanio := currentPCB.CPU_reg[instruccionDecodificada[4]]
			var tamanioEnInt int

			tipoActualRegTamanio := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[4]]).String()

			if tipoActualRegTamanio == "uint32" {
				tamanioEnInt = int(Convertir[uint32](tipoActualRegTamanio, tamanio))
			} else {
				tamanioEnInt = int(Convertir[uint8](tipoActualRegTamanio, tamanio))
			}

			puntero := currentPCB.CPU_reg[instruccionDecodificada[5]]
			var punteroEnInt int

			tipoActualRegPuntero := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[5]]).String()

			if tipoActualRegPuntero == "uint32" {
				punteroEnInt = int(Convertir[uint32](tipoActualRegPuntero, puntero))
			} else {
				punteroEnInt = int(Convertir[uint8](tipoActualRegPuntero, puntero))
			}

			direccionesFisicas := mmu.ObtenerDireccionesFisicas(direccionEnInt, tamanioEnInt, int(currentPCB.PID))

			if cond {

				var fsCreateBody = struct {
					InterfaceName string
					FileName      string
					Address       []globals.DireccionTamanio
					Size          int
					Pointer       int
					Operation     string
				}{
					InterfaceName: instruccionDecodificada[1],
					FileName:      nombre_archivo,
					Address:       direccionesFisicas,
					Size:          tamanioEnInt,
					Pointer:       punteroEnInt,
					Operation:     "WRITE",
				}

				SendIOData(fsCreateBody, "iodata-dialfs")
				currentPCB.EvictionReason = "BLOCKED_IO_DIALFS"
			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true
		/*IO_FS_READ (Interfaz, Nombre Archivo, Registro Dirección, Registro Tamaño, Registro Puntero Archivo):
		  Esta instrucción solicita al Kernel que mediante la interfaz seleccionada, se lea desde el archivo a
		  partir del valor del Registro Puntero Archivo la cantidad de bytes indicada por Registro Tamaño y
		  se escriban en la Memoria a partir de la dirección lógica indicada en el Registro Dirección.*/
	case "IO_FS_READ":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "DIALFS")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO Genéricas")
			currentPCB.EvictionReason = "EXIT"
		} else {
			nombre_archivo := instruccionDecodificada[2]

			direccion := currentPCB.CPU_reg[instruccionDecodificada[3]]
			var direccionEnInt int

			tipoActualRegDireccion := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[3]]).String()

			if tipoActualRegDireccion == "uint32" {
				direccionEnInt = int(Convertir[uint32](tipoActualRegDireccion, direccion))
			} else {
				direccionEnInt = int(Convertir[uint8](tipoActualRegDireccion, direccion))
			}

			tamanio := currentPCB.CPU_reg[instruccionDecodificada[4]]
			var tamanioEnInt int

			tipoActualRegTamanio := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[4]]).String()

			if tipoActualRegTamanio == "uint32" {
				tamanioEnInt = int(Convertir[uint32](tipoActualRegTamanio, tamanio))
			} else {
				tamanioEnInt = int(Convertir[uint8](tipoActualRegTamanio, tamanio))
			}

			puntero := currentPCB.CPU_reg[instruccionDecodificada[5]]
			var punteroEnInt int

			tipoActualRegPuntero := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[5]]).String()

			if tipoActualRegPuntero == "uint32" {
				punteroEnInt = int(Convertir[uint32](tipoActualRegPuntero, puntero))
			} else {
				punteroEnInt = int(Convertir[uint8](tipoActualRegPuntero, puntero))
			}

			direccionesFisicas := mmu.ObtenerDireccionesFisicas(direccionEnInt, tamanioEnInt, int(currentPCB.PID))

			if cond {

				var fsCreateBody = struct {
					InterfaceName string
					FileName      string
					Address       []globals.DireccionTamanio
					Size          int
					Pointer       int
					Operation     string
				}{
					InterfaceName: instruccionDecodificada[1],
					FileName:      nombre_archivo,
					Address:       direccionesFisicas,
					Size:          tamanioEnInt,
					Pointer:       punteroEnInt,
					Operation:     "READ",
				}

				SendIOData(fsCreateBody, "iodata-dialfs")
				currentPCB.EvictionReason = "BLOCKED_IO_DIALFS"
			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true

	case "IO_GEN_SLEEP":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "GENERICA")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO Genéricas")
			currentPCB.EvictionReason = "EXIT"
		} else {
			tiempo_esp, err := strconv.Atoi(instruccionDecodificada[2])
			if err != nil {
				fmt.Print("Error al convertir el tiempo de espera a entero")
			}
			if cond {

				var genSleepBody = struct {
					InterfaceName string
					SleepTime     int
				}{
					InterfaceName: instruccionDecodificada[1],
					SleepTime:     tiempo_esp,
				}

				SendIOData(genSleepBody, "iodata-gensleep")
				currentPCB.EvictionReason = "BLOCKED_IO_GEN"
			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true

	case "IO_STDIN_READ":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "STDIN")
	
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO de lectura")
			currentPCB.EvictionReason = "EXIT"
		} else {
			if cond {
				// Obtener la dirección de memoria desde el registro
				memoryAddress := currentPCB.CPU_reg[instruccionDecodificada[2]]
				tipoActualReg2 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[2]]).String()
				memoryAddressInt := int(Convertir[uint32](tipoActualReg2, memoryAddress))

				// Obtener la cantidad de datos a leer desde el registro
				dataSize := currentPCB.CPU_reg[instruccionDecodificada[3]]
				tipoActualReg3 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[3]]).String()
				dataSizeInt := int(Convertir[uint32](tipoActualReg3, dataSize))

				direccionesFisicas := mmu.ObtenerDireccionesFisicas(memoryAddressInt, dataSizeInt, int(currentPCB.PID))

				var stdinreadBody = struct {
					DireccionesFisicas []globals.DireccionTamanio
					InterfaceName      string
					Tamanio            int
				}{
					DireccionesFisicas: direccionesFisicas,
					InterfaceName:      instruccionDecodificada[1],
					Tamanio:            dataSizeInt,
				}

				SendIOData(stdinreadBody, "iodata-stdin")
				currentPCB.EvictionReason = "BLOCKED_IO_STDIN"

			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true

	case "IO_STDOUT_WRITE":
		cond, err := HallarInterfaz(instruccionDecodificada[1], "STDOUT")
		if err != nil {
			fmt.Print("La interfaz no existe o no acepta operaciones de IO de escritura")
			currentPCB.EvictionReason = "EXIT"
		} else {
			if cond {
				// Obtener la dirección de memoria desde el registro
				memoryAddress := currentPCB.CPU_reg[instruccionDecodificada[2]]
				tipoActualReg2 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[2]]).String()
				memoryAddressInt := int(Convertir[uint32](tipoActualReg2, memoryAddress))

				// Obtener la cantidad de datos a leer desde el registro
				dataSize := currentPCB.CPU_reg[instruccionDecodificada[3]]
				tipoActualReg3 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[3]]).String()
				dataSizeInt := int(Convertir[uint32](tipoActualReg3, dataSize))

				direccionesFisicas := mmu.ObtenerDireccionesFisicas(memoryAddressInt, dataSizeInt, int(currentPCB.PID))

				var stdoutBody = struct {
					DireccionesFisicas []globals.DireccionTamanio
					InterfaceName      string
				}{
					DireccionesFisicas: direccionesFisicas,
					InterfaceName:      instruccionDecodificada[1],
				}

				SendIOData(stdoutBody, "iodata-stdout")
				currentPCB.EvictionReason = "BLOCKED_IO_STDOUT"

			} else {
				currentPCB.EvictionReason = "EXIT"
			}
		}
		pcb.EvictionFlag = true

	case "JNZ":
		if currentPCB.CPU_reg[instruccionDecodificada[1]] != 0 {
			currentPCB.PC = ConvertirUint32(instruccionDecodificada[2])
			currentPCB.CPU_reg["PC"] = uint32(currentPCB.PC)
		}

	case "SET":
		tipoReg := pcb.TipoReg(instruccionDecodificada[1])
		valor := instruccionDecodificada[2]

		if tipoReg == "uint32" {
			currentPCB.CPU_reg[instruccionDecodificada[1]] = ConvertirUint32(valor)
		} else {
			currentPCB.CPU_reg[instruccionDecodificada[1]] = ConvertirUint8(valor)
		}

		if instruccionDecodificada[1] == "PC" {
			currentPCB.PC = ConvertirUint32(valor)
		}

	case "SUM":
		tipoReg1 := pcb.TipoReg(instruccionDecodificada[1])
		valorReg2 := currentPCB.CPU_reg[instruccionDecodificada[2]]

		tipoActualReg1 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[1]]).String()
		tipoActualReg2 := reflect.TypeOf(valorReg2).String()

		if tipoReg1 == "uint32" {
			currentPCB.CPU_reg[instruccionDecodificada[1]] = Convertir[uint32](tipoActualReg1, currentPCB.CPU_reg[instruccionDecodificada[1]]) + Convertir[uint32](tipoActualReg2, valorReg2)

		} else {
			currentPCB.CPU_reg[instruccionDecodificada[1]] = Convertir[uint8](tipoActualReg1, currentPCB.CPU_reg[instruccionDecodificada[1]]) + Convertir[uint8](tipoActualReg2, valorReg2)
		}

		if instruccionDecodificada[1] == "PC" {
			currentPCB.PC = currentPCB.CPU_reg["PC"].(uint32)
		}

	case "SUB":
		//SUB (Registro Destino, Registro Origen): Resta al Registro Destino
		//el Registro Origen y deja el resultado en el Registro Destino.
		tipoReg1 := pcb.TipoReg(instruccionDecodificada[1])
		valorReg2 := currentPCB.CPU_reg[instruccionDecodificada[2]]

		tipoActualReg1 := reflect.TypeOf(currentPCB.CPU_reg[instruccionDecodificada[1]]).String()
		tipoActualReg2 := reflect.TypeOf(valorReg2).String()

		if tipoReg1 == "uint32" {
			currentPCB.CPU_reg[instruccionDecodificada[1]] = Convertir[uint32](tipoActualReg1, currentPCB.CPU_reg[instruccionDecodificada[1]]) - Convertir[uint32](tipoActualReg2, valorReg2)

		} else {
			currentPCB.CPU_reg[instruccionDecodificada[1]] = Convertir[uint8](tipoActualReg1, currentPCB.CPU_reg[instruccionDecodificada[1]]) - Convertir[uint8](tipoActualReg2, valorReg2)
		}

		if instruccionDecodificada[1] == "PC" {
			currentPCB.PC = currentPCB.CPU_reg["PC"].(uint32)
		}

	case "WAIT":
		currentPCB.RequestedResource = instruccionDecodificada[1]
		fmt.Print("Requested Resource: ", currentPCB.RequestedResource+"\n") // *Lo hace bien
		currentPCB.EvictionReason = "WAIT"
		pcb.EvictionFlag = true

	case "SIGNAL":
		currentPCB.RequestedResource = instruccionDecodificada[1]
		currentPCB.EvictionReason = "SIGNAL"
		pcb.EvictionFlag = true

	case "MOV_OUT":
		//MOV_OUT(Registro Dirección, Registro Datos): Lee el valor del Registro Datos y lo escribe en la dirección física de memoria
		//obtenida a partir de la Dirección Lógica almacenada en el Registro Dirección.

		// Leer el valor y tamaño del registro de datos (2)
		var tamanio2 int
		tipoReg2 := pcb.TipoReg(instruccionDecodificada[2])
		if tipoReg2 == "uint32" {
			tamanio2 = 4
		} else if tipoReg2 == "uint8" {
			tamanio2 = 1
		}

		// Leer la dirección lógica del registro de dirección (1)
		valorReg1 := currentPCB.CPU_reg[instruccionDecodificada[1]]
		tipoActualReg1 := reflect.TypeOf(valorReg1).String()

		direc_log := Convertir[uint32](tipoActualReg1, valorReg1)

		direcsFisicas := mmu.ObtenerDireccionesFisicas(int(direc_log), tamanio2, int(currentPCB.PID))

		valorReg2 := currentPCB.CPU_reg[instruccionDecodificada[2]]
		tipoActualReg2 := reflect.TypeOf(valorReg2).String()

		var valorEnBytes []byte

		if tipoReg2 == "uint32" {
			valor2EnUint := Convertir[uint32](tipoActualReg2, valorReg2)
			valorEnBytes = []byte{byte(valor2EnUint)} 

		} else {
			valor2EnUint := Convertir[uint8](tipoActualReg2, valorReg2) 
			valorEnBytes = []byte{valor2EnUint}
		}
		valorEnBytesRelleno := make([]byte, tamanio2)
		inicio := len(valorEnBytesRelleno) - len(valorEnBytes)
		// Realizar la copia en la posición calculada
		copy(valorEnBytesRelleno[inicio:], valorEnBytes)

		solicitudesmemoria.SolicitarEscritura(direcsFisicas, valorEnBytesRelleno, int(currentPCB.PID)) //([direccion fisica y tamanio], valorAEscribir, pid

		//----------------------------------------------------------------------------

		// MOV_IN (Registro Datos, Registro Dirección): Lee el valor
		// de memoria correspondiente a la Dirección Lógica que se encuentra
		// en el Registro Dirección y lo almacena en el Registro Datos.

	case "MOV_IN":

		var tamanio int
		tipoReg1 := pcb.TipoReg(instruccionDecodificada[1])
		if tipoReg1 == "uint32" {
			tamanio = 4
		} else if tipoReg1 == "uint8" {
			tamanio = 1
		}

		valorReg2 := currentPCB.CPU_reg[instruccionDecodificada[2]]
		tipoActualReg2 := reflect.TypeOf(valorReg2).String()

		direc_log := Convertir[uint32](tipoActualReg2, valorReg2)

		fmt.Println("El valor de la direc logica es", int(direc_log))

		// Obtenemos la direcion fisica del reg direccion
		direcsFisicas := mmu.ObtenerDireccionesFisicas(int(direc_log), tamanio, int(currentPCB.PID))

		fmt.Println("Direcciones fisicas: ", direcsFisicas)

		//Obtenemos el valor guardado en las direcciones fisicas
		datos := solicitudesmemoria.SolicitarLectura(direcsFisicas, int(currentPCB.PID))
		fmt.Println("Los datos obtenidos de memoria son: ", datos)
		
		// Almacenamos lo leido en el registro destino
		var datosAAlmacenar uint64

		if tipoReg1 == "uint32" {

			bigInt := big.NewInt(0).SetBytes(datos)
			datosAAlmacenar = bigInt.Uint64()

			currentPCB.CPU_reg[instruccionDecodificada[1]] = uint32(datosAAlmacenar)

		} else {
			
			datosAAlmacenar = uint64(datos[0])
			currentPCB.CPU_reg[instruccionDecodificada[1]] = uint8(datosAAlmacenar)
		}

		//-----------------------------------------------------------------------------
		//COPY_STRING (Tamaño): Toma del string apuntado por el registro SI y
		//copia la cantidad de bytes indicadas en el parámetro tamaño a la
		//posición de memoria apuntada por el registro DI.

	case "COPY_STRING":
		tamanio := globals.PasarAInt(instruccionDecodificada[1])
		//Buscar la direccion logica del registro SI
		valorRegSI := currentPCB.CPU_reg["SI"]
		tipoActualRegSI := reflect.TypeOf(valorRegSI).String()
		var direc_logicaSI int
		
		if tipoActualRegSI == "uint32" {
			valorSIConv := Convertir[uint32](tipoActualRegSI, valorRegSI)
			direc_logicaSI = int(valorSIConv)

		} else {
			valorSIConv := Convertir[uint8](tipoActualRegSI, valorRegSI)
			direc_logicaSI = int(valorSIConv)
		}

		direcsFisicasSI := mmu.ObtenerDireccionesFisicas(direc_logicaSI, tamanio, int(currentPCB.PID))

		// Lee lo que hay en esa direccion fisica pero no todo, lees lo que te pasaron x param
		datos := solicitudesmemoria.SolicitarLectura(direcsFisicasSI, int(currentPCB.PID))
		fmt.Println("Los datos leidos de memoria son: ", datos)

		// Busca la direccion logica del registro DI
		valorRegDI := currentPCB.CPU_reg["DI"]
		tipoActualRegDI := reflect.TypeOf(valorRegDI).String()
		direc_logicaDI := int(Convertir[uint32](tipoActualRegDI, valorRegDI))

		// Obtiene la direccion Fisica asociada
		direcsFisicasDI := mmu.ObtenerDireccionesFisicas(direc_logicaDI, tamanio, int(currentPCB.PID))
		
		valorEnBytesRelleno := make([]byte, tamanio)
		inicio := len(valorEnBytesRelleno) - len(datos)
		// Realizar la copia en la posición calculada
		copy(valorEnBytesRelleno[inicio:], datos)		// Carga en esa direccion fisica lo que leiste antes
		solicitudesmemoria.SolicitarEscritura(direcsFisicasDI, valorEnBytesRelleno, int(currentPCB.PID)) //([direccion fisica y tamanio], valorAEscribir, pid)

	//RESIZE (Tamaño)
	case "RESIZE":
		tamanio := globals.PasarAInt(instruccionDecodificada[1])
		
		respuestaResize := solicitudesmemoria.Resize(tamanio)
		fmt.Println("el resize devuelve", respuestaResize)
		if respuestaResize != "\"OK\"" {
			currentPCB.EvictionReason = "OUT_OF_MEMORY"
			pcb.EvictionFlag = true
		}
	}
}

type Uint interface{ ~uint8 | ~uint32 }

func Convertir[T Uint](tipo string, parametro interface{}) T {

	if parametro == "" {
		fmt.Printf("La cadena de texto está vacía")
	}

	fmt.Print("El tipo a convertir es: ", tipo)

	switch tipo {
	case "uint8":
		valor := parametro.(uint8)
		return T(valor)
	case "uint32":
		valor := parametro.(uint32)
		return T(valor)
	case "float64":
		valor := parametro.(float64)
		return T(valor)
	case "int":
		valor := parametro.(int)
		return T(valor)
	}
	return T(0)
}

type SearchInterface struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func HallarInterfaz(nombre string, tipo string) (bool, error) {
	interf := SearchInterface{
		Name: nombre,
		Type: tipo,
	}

	fmt.Println("Interfaz a buscar: ", interf)

	jsonData, err := json.Marshal(interf)
	if err != nil {
		return false, fmt.Errorf("failed to encode interface: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/io-interface", globals.Configcpu.IP_kernel, globals.Configcpu.Port_kernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("POST request failed. Failed to send interface: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	var response bool
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return false, fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Println("Interfaz encontrada")

	return response, nil
}

/*
	 SendIOData: Comunica la información necesaria a kernel para el uso de cualquier body de interfaz de entrada/salida

	 @param datum: Estructura con la información necesaria para la comunicación (La estructura usada va a depender de la interfaz a utilizar)
	 @param endpoint: Endpoint al que se va a enviar la información
		- "iodata-gensleep"
		- "iodata-stdin"
		- "iodata-stdout"
		- "iodata-dialfs"
	 @return error: Error en caso de que la comunicación falle

*
*/
func SendIOData(datum interface{}, endpoint string) error {
	jsonData, err := json.Marshal(datum)
	if err != nil {
		return fmt.Errorf("failed to encode interface: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/%s", globals.Configcpu.IP_kernel, globals.Configcpu.Port_kernel, endpoint)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("POST request failed. Failed to send interface: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	return nil
}

func ConvertirUint8(parametro string) uint8 {
	parametroConvertido, err := strconv.Atoi(parametro)
	if err != nil {
		fmt.Print("Error al convertir el parametro a uint8")
	}
	return uint8(parametroConvertido)
}

func ConvertirUint32(parametro string) uint32 {
	parametroConvertido, err := strconv.Atoi(parametro)
	if err != nil {
		fmt.Print("Error al convertir el parametro a uint32")
	}
	return uint32(parametroConvertido)
}
