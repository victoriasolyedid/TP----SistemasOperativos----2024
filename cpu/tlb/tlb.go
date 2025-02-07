package tlb

import (
	"fmt"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
)

type Pagina_marco struct {
	Pagina int
	Marco  int
}

type TLB []map[int]Pagina_marco

var CurrentTLB TLB
var OrderedKeys []int //mantiene el orden de las claves en la TLB

func BuscarEnTLB(pid, pagina int) bool {
	if globals.Configcpu.Number_felling_tlb > 0 {
		for _, entradaTLB := range CurrentTLB {
			if entry, exists := entradaTLB[pid]; exists && entry.Pagina == pagina {
				return true
			}
		}
	}
	return false
}

func FrameEnTLB(pid int, pagina int) int {
	if globals.Configcpu.Number_felling_tlb > 0 {
		for _, entradaTLB := range CurrentTLB {
			if entry, exists := entradaTLB[pid]; exists && entry.Pagina == pagina {
				ActualizarTLB(pid, pagina, entry.Marco)
				return entry.Marco
			}
		}
	}
	return -1
}

func ObtenerPagina(direccionLogica int, nroPag int, tamanio int) int {
	pagina := (direccionLogica + nroPag*tamanio) / tamanio

	return pagina
}

func ObtenerOffset(direccionLogica int, nroPag int, tamanio int) int {

	offset := (direccionLogica + nroPag*tamanio) % tamanio

	return offset
}

func CalcularDireccionFisica(frame int, offset int, tamanio int) int {

	direccionBase := frame * tamanio

	return direccionBase + offset

}

func ActualizarTLB(pid, pagina, marco int) {

	if globals.Configcpu.Number_felling_tlb > 0 {
		switch globals.Configcpu.Algorithm_tlb {
		case "FIFO":
			if !BuscarEnTLB(pid, pagina) { //Si la página no está en la tlb
				if len(CurrentTLB) < globals.Configcpu.Number_felling_tlb {
					nuevoElemento := map[int]Pagina_marco{
						pid: {Pagina: pagina, Marco: marco},
					}
					CurrentTLB = append(CurrentTLB, nuevoElemento)
					fmt.Printf("Se agregó la entrada %d a la TLB", CurrentTLB)
					fmt.Println("LA TLB QUEDO ASI: ")
					for i := range CurrentTLB {
						fmt.Println(CurrentTLB[i])
					}
				} else {
					// Remover el primer elemento (FIFO) y agregar el nuevo
					CurrentTLB = append(CurrentTLB[1:], map[int]Pagina_marco{
						pid: {Pagina: pagina, Marco: marco},
					})
					fmt.Printf("Se agregó la entrada %d a la TLB", CurrentTLB)
					fmt.Println("LA TLB QUEDO ASI: ")
					for i := range CurrentTLB {
						fmt.Println(CurrentTLB[i])
					}
				}
			}

		case "LRU":
			/**Lista “jenga” con números de págs -> con cada referencia se coloca (o se mueve, si ya existe) la pág al final de la lista.
			Se elige como víctima la primera de la lista.*/
			if !BuscarEnTLB(pid, pagina) { // La página no está en la TLB
				if len(CurrentTLB) < globals.Configcpu.Number_felling_tlb { // Hay lugar en la TLB
					CurrentTLB = append(CurrentTLB, map[int]Pagina_marco{pid: {Pagina: pagina, Marco: marco}})
				} else { // No hay lugar en la TLB, se reemplaza la página menos recientemente utilizada
					CurrentTLB = append(CurrentTLB[1:], map[int]Pagina_marco{pid: {Pagina: pagina, Marco: marco}})
				}
			} else { // La página está en la TLB, se mueve al final de la lista
				var indice int
				for i, entrada := range CurrentTLB {
					if entrada[pid].Pagina == pagina {
						indice = i // indica el valor de la lista de mapas en donde se encuentra la pagina
						break
					}
				}
				CurrentTLB = append(CurrentTLB[:indice], CurrentTLB[indice+1:]...)
				CurrentTLB = append(CurrentTLB, map[int]Pagina_marco{pid: {Pagina: pagina, Marco: marco}})
			}

			// Imprimir la TLB
			fmt.Println("LA TLB QUEDO ASI: ")
			for i := range CurrentTLB {
				fmt.Println(CurrentTLB[i])
			}
		}
	}
}

func ActualizarOrdenDeAcceso(pid, pagina, marco int) {
	// Elimina la clave si ya existe
	for i, key := range OrderedKeys {
		if key == pid {
			OrderedKeys = append(OrderedKeys[:i], OrderedKeys[i+1:]...)
			break
		}
	}
	// Añade la clave al final (más recientemente utilizada)
	OrderedKeys = append(OrderedKeys, pid)

	// Actualizar o agregar la entrada en CurrentTLB
	encontrado := false
	for _, entrada := range CurrentTLB {
		if entrada[pid].Pagina == pagina && entrada[pid].Marco == marco {
			encontrado = true
			break
		}
	}
	if !encontrado {
		nuevoElemento := map[int]Pagina_marco{
			pid: {Pagina: pagina, Marco: marco},
		}
		CurrentTLB = append(CurrentTLB, nuevoElemento)
		fmt.Printf("Se agregó la entrada %d a la TLB\n", pid)
		fmt.Println("LA TLB QUEDO ASI: ")
		for i := range CurrentTLB {
			fmt.Println(CurrentTLB[i])
		}
	}
}
