package globals

import "sync"

// Global variables:
var InstruccionesProceso = make(map[int][]string)

// Global semaphores
var (
	InstructionsMutex sync.Mutex
)

type T_ConfigMemory struct {
	Port              int    `json:"port"`
	Memory_size       int    `json:"memory_size"`
	Page_size         int    `json:"page_size"`
	Instructions_path string `json:"instructions_path"`
	Delay_response    int    `json:"delay_response"`
}

var Configmemory *T_ConfigMemory

var CurrentBitMap []int
var Frames int

type Frame int

// Tabla de páginas (donde a cada página(indice) le corresponde un frame)
type TablaPaginas []Frame

// Diccionario para identificar a que proceso pertenece cada TablaPaginas
var Tablas_de_paginas map[int]TablaPaginas

// Inicializo la memoria
var User_Memory []byte // de 0 a 15 corresponde a una página, marco compuesto por 16 bytes (posiciones)

type DireccionTamanio struct {
	DireccionFisica int
	Tamanio         int
}
