package IO_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-golang/entradasalida/globals"
	ioutils "github.com/sisoputnfrba/tp-golang/entradasalida/utils"
)

/**
 * InicializarFS: inicializa el sistema de archivos (crea el directorio y el archivo de bitmap) o carga el sistema de archivos desde disco (bitmap.dat y bloques.dat)
 */
func InicializarFS() {
	// Define el nombre del directorio y los archivos
	dirName := "dialfs"
	bitmapFile := "bitmap.dat"
	blocksFile := "bloques.dat"

	// Crea el directorio si no existe
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err = os.Mkdir(dirName, 0755)
		if err != nil {
			fmt.Println("Failed creating directory: ", err)
		}

		fmt.Println("FS - Directorio CREADO")
	}

	// Crea el archivo de bitmap si no existe
	if _, err := os.Stat(dirName + "/" + bitmapFile); os.IsNotExist(err) {
		// Carga el bitmap en globals
		globals.CurrentBitMap = ioutils.NewBitMap(globals.ConfigIO.Dialfs_block_count)
		ioutils.ActualizarBitmap()

		fmt.Println("FS - Bitmap.dat CREADO")

	} else {
		// Carga el contenido del archivo
		file, err := os.Open(dirName + "/" + bitmapFile)
		if err != nil {
			fmt.Println("Failed opening file: ", err)
		}
		defer file.Close()

		// Lee el contenido del archivo
		contenidoBitmap, err := os.ReadFile(dirName + "/" + bitmapFile)
		if err != nil {
			fmt.Println("Failed to read file: ", err)
		}

		globals.CurrentBitMap = contenidoBitmap

		// Verifica que el tamaño del bitmap sea el correcto
		if len(globals.CurrentBitMap) != globals.ConfigIO.Dialfs_block_count {
			fmt.Println("Bitmap size is incorrect")
		}

		fmt.Println("FS - Bitmap.dat LEIDO")
	}

	// Crea el archivo de bloques si no existe
	if _, err := os.Stat(dirName + "/" + blocksFile); os.IsNotExist(err) {
		// Carga los bloques en globals
		globals.Blocks = make([]byte, globals.ConfigIO.Dialfs_block_count*globals.ConfigIO.Dialfs_block_size)
		ioutils.ActualizarBloques()

		fmt.Println("FS - Bloques.dat CREADO")

	} else {
		// Carga el contenido del archivo
		file, err := os.Open(dirName + "/" + blocksFile)
		if err != nil {
			fmt.Println("Failed opening file: ", err)
		}
		defer file.Close()

		// Lee el contenido del archivo
		contenidoBloques, err := os.ReadFile(dirName + "/" + blocksFile)
		if err != nil {
			fmt.Println("Failed to read file: ", err)
		}

		globals.Blocks = contenidoBloques

		// Verifica que el tamaño del slice de bloques sea el correcto
		if len(globals.Blocks) != globals.ConfigIO.Dialfs_block_count*globals.ConfigIO.Dialfs_block_size {
			fmt.Println("Blocks slice size is incorrect")
		}

		fmt.Println("FS - Bloques.dat LEIDO")
	}

	// Carga los archivos metadata del directorio si existen
	globals.Fcbs = make(map[string]globals.Metadata)
	files, err := os.ReadDir(dirName)
	if err != nil {
		fmt.Println("Failed reading directory: ", err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".txt") {
			continue
		}
		archivo := ioutils.LeerArchivoEnStruct(dirName + "/" + file.Name())
		globals.Fcbs[file.Name()] = *archivo

		fmt.Println("FS - Archivo CARGADO: ", file.Name())
	}
}

// * Funciones para ciclo de instrucción

/**
 * CreateFile: se define un nuevo archivo y se lo posiciona en el sistema de archivos (se crea el FCB y se lo agrega al directorio)
 */
func CreateFile(pid int, nombreArchivo string) {
	//Se crea el archivo metadata, con el size en 0 y 1 bloque asignado
	//Archivos de metadata en el módulo FS cargados en alguna estructura para que les sea fácil acceder

	bloqueInicial := ioutils.CalcularBloqueLibre()
	ioutils.Set(bloqueInicial - 1)

	// Crea la metadata
	metadata := globals.Metadata{
		InitialBlock: bloqueInicial,
		Size:         0,
	}

	// Convierte la metadata a JSON
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		fmt.Println("Failed to marshal metadata: ", err)
	}

	// Crea el archivo
	ioutils.CrearModificarArchivo(nombreArchivo, metadataJson)

	// Agrego el FCB al directorio
	globals.Fcbs[nombreArchivo] = metadata
	log.Printf("PID: %d - Crear Archivo: %s", pid, nombreArchivo)
}

/**
 * DeleteFile: elimina un archivo del sistema de archivos y su FCB asociado (incluye liberar los bloques de datos)
 */
func DeleteFile(pid int, nombreArchivo string) error {
	rutaArchivo := "dialfs/" + nombreArchivo

	// Paso 1: Verificar si el archivo existe
	if _, err := os.Stat(rutaArchivo); os.IsNotExist(err) {
		// El archivo no existe
		return errors.New("el archivo no existe")
	}

	// Leer la información del archivo antes de eliminarlo
	archivo := ioutils.LeerArchivoEnStruct(rutaArchivo)

	sizeArchivo := archivo.Size
	sizeArchivoEnBloques := int(math.Max(1, math.Ceil(float64(sizeArchivo)/float64(globals.ConfigIO.Dialfs_block_size))))

	posBloqueInicial := archivo.InitialBlock - 1

	// Paso 2: Eliminar el archivo del sistema de archivos
	err := os.Remove(rutaArchivo)
	if err != nil {
		// Error al intentar eliminar el archivo
		return err
	}

	// Paso 3: Eliminar el FCB asociado y liberar los bloques de datos
	delete(globals.Fcbs, nombreArchivo)

	for i := 0; i < sizeArchivoEnBloques; i++ {
		ioutils.Clear(posBloqueInicial + i)
	}

	ioutils.ActualizarBitmap()

	log.Printf("PID: %d - Eliminar Archivo: %s", pid, nombreArchivo)

	return nil
}

/*
*
  - ReadFile: Lee un archivo del sistema de archivos
    // Leer un bloque de fs implica escribirlo en memoria
    (Interfaz, Nombre Archivo, Registro Dirección, Registro Tamaño, Registro Puntero Archivo):
    Esta instrucción solicita al Kernel que mediante la interfaz seleccionada, se lea desde el
    archivo a partir del valor del Registro Puntero Archivo la cantidad de bytes indicada por Registro
    Tamaño y se escriban en la Memoria a partir de la dirección lógica indicada en el Registro Dirección.
*/
func ReadFile(pid int, nombreArchivo string, direccionesFisicas []globals.DireccionTamanio, tamanioLeer int, puntero int) {
	var contenidoALeer []byte

	posBloqueInicial := globals.Fcbs[nombreArchivo].InitialBlock - 1
	primerByteArchivo := posBloqueInicial * globals.ConfigIO.Dialfs_block_size

	posicionPuntero := primerByteArchivo + puntero
	limiteLeer := tamanioLeer + posicionPuntero

	limiteArchivo := primerByteArchivo + globals.Fcbs[nombreArchivo].Size

	// Verificar tamaño del archivo a leer valido
	if limiteLeer > limiteArchivo {
		fmt.Println("El tamaño a leer es superior a el correspondiente del archivo")
	} else {
		contenidoALeer = ioutils.ReadFs(nombreArchivo, puntero, tamanioLeer)
		IO_DIALFS_READ(pid, direccionesFisicas, contenidoALeer)
	}

	log.Printf("PID: %d - Leer Archivo: %s - Tamaño a Leer: %d - Puntero Archivo: %d", pid, nombreArchivo, tamanioLeer, puntero)
}

/*
*
  - WriteFile: Escribe un archivo del sistema de archivos
    IO_FS_WRITE (Interfaz, Nombre Archivo, Registro Dirección, Registro Tamaño, Registro Puntero Archivo):
    Esta instrucción solicita al Kernel que mediante la interfaz seleccionada, se lea desde Memoria
    la cantidad de bytes indicadas por el Registro Tamaño a partir de la dirección lógica que se encuentra
    en el Registro Dirección y se escriban en el archivo a partir del valor del Registro Puntero Archivo.
  - Escritura de un bloque de fs implica leerlo de memoria para luego escribirlo en fs
*/
func WriteFile(pid int, nombreArchivo string, direcciones []globals.DireccionTamanio, tamanio int, puntero int) {
	archivo := globals.Fcbs[nombreArchivo]

	bloqueInicial := archivo.InitialBlock
	posBloqueInicial := bloqueInicial - 1
	primerByteArchivo := posBloqueInicial * globals.ConfigIO.Dialfs_block_size
	ultimoByteArchivo := primerByteArchivo + archivo.Size

	posicionPuntero := primerByteArchivo + puntero

	leidoEnMemoria := IO_DIALFS_WRITE(pid, direcciones)
	cantidadBytesLeidos := len(leidoEnMemoria)

	posInicialByteAEscribir := posicionPuntero + cantidadBytesLeidos

	cantidadBytesFinales := math.Max(float64(archivo.Size), float64(cantidadBytesLeidos+puntero))

	if posInicialByteAEscribir > ultimoByteArchivo {
		TruncateFile(pid, nombreArchivo, int(cantidadBytesFinales))
	}

	fmt.Println("CONTENIDO A ESCRIBIR EN FS (", nombreArchivo, "): ", leidoEnMemoria)
	fmt.Println("CONTENIDO EN STRING (", nombreArchivo, "): ", string(leidoEnMemoria))

	ioutils.WriteFs(leidoEnMemoria, posicionPuntero)

	log.Printf("PID: %d - Escribir Archivo: %s - Tamaño a Escribir: %d - Puntero Archivo: %d", pid, nombreArchivo, tamanio, puntero)
}

/**
 * TruncateFile: Trunca un archivo del sistema de archivos (puede incluir compactar el archivo)
 */
func TruncateFile(pid int, nombreArchivo string, tamanioDeseado int) { //revisar si tiene que devolver un msje
	archivo := globals.Fcbs[nombreArchivo]

	bloqueInicial := archivo.InitialBlock
	//posBloqueInicial := bloqueInicial - 1

	tamOriginalEnBloques := int(math.Max(1, math.Ceil(float64(archivo.Size)/float64(globals.ConfigIO.Dialfs_block_size))))
	tamFinalEnBloques := int(math.Ceil(float64(tamanioDeseado) / float64(globals.ConfigIO.Dialfs_block_size)))

	var tamanioATruncarEnBloques int
	if tamanioDeseado > archivo.Size {
		tamanioATruncarEnBloques = tamFinalEnBloques - tamOriginalEnBloques
	} else {
		tamanioATruncarEnBloques = tamOriginalEnBloques - tamFinalEnBloques
	}

	fmt.Println("El tamaño original de ", nombreArchivo, " en bloques es ", tamOriginalEnBloques)

	// bloqueFinalInicial: El final del archivo actual, bloque donde tiene que arrancar el siguiente archivo
	bloqueFinalInicial := bloqueInicial + tamOriginalEnBloques

	fmt.Println("Bloque inicial ", nombreArchivo, ": ", bloqueInicial)
	fmt.Println("Bloque final ", nombreArchivo, ": ", bloqueFinalInicial)

	// Chequeamos si el archivo tiene que crecer o achicarse
	// Si el archivo crece
	if tamanioDeseado > archivo.Size {
		contenidoArchivo := ioutils.ReadFs(nombreArchivo, 0, -1)
	
		fmt.Println("Tamaño a agrandar ", nombreArchivo, " en bloques: ", tamanioATruncarEnBloques)
		bloquesLibresAlFinalDelArchivo := ioutils.CalcularBloquesLibreAPartirDe(bloqueFinalInicial)

		// Hay bloques libres al final del archivo
		if bloquesLibresAlFinalDelArchivo >= tamanioATruncarEnBloques {
			fmt.Println("FS - Hay bloques libres al final de ", nombreArchivo)

			ioutils.OcuparBloquesDesde(bloqueFinalInicial, tamanioATruncarEnBloques)
			archivo.Size = tamanioDeseado

			archivoMarshallado, err := json.Marshal(archivo)
			if err != nil {
				fmt.Println("Failed to marshal metadata: ", err)
			}

			ioutils.CrearModificarArchivo(nombreArchivo, archivoMarshallado)
			globals.Fcbs[nombreArchivo] = archivo

			// No hay bloques libres al final del archivo
		} else if bloquesLibresAlFinalDelArchivo < tamanioATruncarEnBloques {
			// Buscar si entra en algún lugar el archivo completo
			posBloqueEnElQueEntro := ioutils.EntraEnDisco(tamFinalEnBloques)

			// Entra en algún lugar el archivo completo todo junto
			if posBloqueEnElQueEntro != -1 {
				bloqueEnElQueEntro := posBloqueEnElQueEntro + 1
				fmt.Println(nombreArchivo, " ENTRA A PARTIR DEL BLOQUE ", bloqueEnElQueEntro)
				fmt.Println("FS - ", nombreArchivo, " Entra en algún lugar el archivo completo")
				// Limpiar el bitmap
				ioutils.LiberarBloquesDesde(bloqueInicial, tamOriginalEnBloques)
				archivo.InitialBlock = bloqueEnElQueEntro // !
				archivo.Size = tamanioDeseado
				// Setear en el nuevo lugar
				ioutils.OcuparBloquesDesde(bloqueEnElQueEntro, tamFinalEnBloques)

				byteInicialDestino := bloqueEnElQueEntro * globals.ConfigIO.Dialfs_block_size
				ioutils.WriteFs(contenidoArchivo, byteInicialDestino)

				archivoMarshallado, err := json.Marshal(archivo)
				if err != nil {
					fmt.Println("Failed to marshal metadata: ", err)
				}

				ioutils.CrearModificarArchivo(nombreArchivo, archivoMarshallado)
				globals.Fcbs[nombreArchivo] = archivo

				archivoFinal := ioutils.LeerArchivoEnStruct("dialfs/" + nombreArchivo)
				log.Println("ARCHIVO - ", nombreArchivo, " tiene un tamaño de ", archivoFinal.Size, " y comienza en el bloque ", archivoFinal.InitialBlock)
				log.Println("FCB - ", nombreArchivo, " tiene un tamaño de ", globals.Fcbs[nombreArchivo].Size, " y comienza en el bloque ", globals.Fcbs[nombreArchivo].InitialBlock)

				time.Sleep(time.Duration(globals.ConfigIO.Dialfs_compaction_delay)) //

				// No entra el archivo pero alcanza la cantidad de bloques libres
			} else if ioutils.ContadorDeEspaciosLibres() >= tamFinalEnBloques {
				fmt.Println("FS - ", nombreArchivo, " No entra el archivo pero alcanza la cantidad de bloques libres")
				//Si no entra en ningún lugar, compactar

				delete(globals.Fcbs, nombreArchivo)

				Compactar()

				primerBloqueLibre := ioutils.CalcularBloqueLibre()
				fmt.Println("FS - ", nombreArchivo, " Primer bloque libre: ", primerBloqueLibre)

				archivo.InitialBlock = primerBloqueLibre
				archivo.Size = tamanioDeseado

				posPrimerBloqueLibre := primerBloqueLibre - 1

				ioutils.WriteFs(contenidoArchivo, posPrimerBloqueLibre*globals.ConfigIO.Dialfs_block_size)

				ioutils.OcuparBloquesDesde(primerBloqueLibre, tamFinalEnBloques)

				archivoMarshallado, err := json.Marshal(archivo)
				if err != nil {
					fmt.Println("Failed to marshal metadata: ", err)
				}

				ioutils.CrearModificarArchivo(nombreArchivo, archivoMarshallado)
				globals.Fcbs[nombreArchivo] = archivo

			} else {
				fmt.Print("No hay espacio suficiente para el archivo ", nombreArchivo)
			}
		}

		// Si el archivo se achica
	} else if tamanioDeseado < archivo.Size {
		// Liberar los bloques que ya no se usan
		bloqueFinal := bloqueFinalInicial - 1
		ioutils.LiberarBloque(bloqueFinal, tamanioATruncarEnBloques)

		archivo.Size = tamanioDeseado

		archivoMarshallado, err := json.Marshal(archivo)
		if err != nil {
			fmt.Println("Failed to marshal metadata: ", err)
		}

		ioutils.CrearModificarArchivo(nombreArchivo, archivoMarshallado)
		globals.Fcbs[nombreArchivo] = archivo
	}

	log.Printf("PID: %d - Truncar Archivo: %s - Tamaño: %d", pid, nombreArchivo, tamanioDeseado)
}

func Compactar() {
	var bloquesDeArchivos = make([]byte, len(globals.Blocks))

	tamBloqueEnBytes := globals.ConfigIO.Dialfs_block_size

	for i := range globals.CurrentBitMap {
		ioutils.Clear(i)
	}
	fmt.Println("FS - Bitmap limpio: ", globals.CurrentBitMap)
	fmt.Println("FS - Cant. bloques: ", len(globals.Blocks))

	offset := 0
	for nombreArchivo, metadata := range globals.Fcbs {
		tamArchivoEnBloques := int(math.Max(1, math.Ceil(float64(metadata.Size)/float64(tamBloqueEnBytes))))
		bloqueInicial := metadata.InitialBlock

		fmt.Println("Moviendo archivo: ", nombreArchivo)
		fmt.Println("Bloque inicial: ", bloqueInicial)

		for i := 0; i < tamArchivoEnBloques; i++ {
			primerByteACopiar := (bloqueInicial + i - 1) * tamBloqueEnBytes
			ultimoByteACopiar := primerByteACopiar + tamBloqueEnBytes //- 1

			n := copy(bloquesDeArchivos[offset:], globals.Blocks[primerByteACopiar:ultimoByteACopiar])

			fmt.Println("N vale: ", n)
			fmt.Println("Primer byte a copiar: ", primerByteACopiar)
			fmt.Println("Ultimo byte a copiar: ", ultimoByteACopiar)
			offset += n // Actualiza el offset con el número de bytes copiados
			fmt.Println("Offset vespues vale: ", offset)
		}

		metadata.InitialBlock = ioutils.CalcularBloqueLibre()

		ioutils.OcuparBloquesDesde(metadata.InitialBlock, tamArchivoEnBloques)

		archivoMarshallado, err := json.Marshal(metadata)
		if err != nil {
			fmt.Println("Failed to marshal metadata: ", err)
		}

		ioutils.CrearModificarArchivo(nombreArchivo, archivoMarshallado)
		globals.Fcbs[nombreArchivo] = metadata
	}

	copy(globals.Blocks, bloquesDeArchivos[:len(globals.Blocks)])

	ioutils.ActualizarBloques()
	ioutils.ActualizarBitmap()

	time.Sleep(time.Duration(globals.ConfigIO.Dialfs_compaction_delay))
}
