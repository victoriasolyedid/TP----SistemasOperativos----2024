package ioutils

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/sisoputnfrba/tp-golang/entradasalida/globals"
)

/**
  - CrearModificarArchivo: carga un archivo en el sistema de archivos
  - @param nombreArchivo: nombre del archivo a cargar
  - @param contenido: contenido del archivo a cargar (en bytes)
*/
func CrearModificarArchivo(nombreArchivo string, contenido []byte) {
	var file *os.File
	var err error

	if nombreArchivo != "dialfs/bitmap.dat" && nombreArchivo != "dialfs/bloques.dat" {
		nombreArchivo = "dialfs/" + nombreArchivo
	}

	// Crea un nuevo archivo si no existe
	if _, err := os.Stat(nombreArchivo); os.IsNotExist(err) {
		file, err = os.Create(nombreArchivo)
		if err != nil {
			fmt.Println("Failed creating file: ", err)
		}
	} else {
		file, err = os.OpenFile(nombreArchivo, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Failed opening file: ", err)
		}
	}

	// Cierra el archivo al final de la función
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Println("Failed closing file: ", err)
		}
	}()

	// Escribe el contenido en el archivo
	_, err = file.Write(contenido)
	if err != nil {
		fmt.Println("Failed writing to file: ", err)
	}

	// Guarda los cambios en el archivo
	err = file.Sync()
	if err != nil {
		fmt.Println("Failed syncing file: ", err)
	}
}

func LeerArchivoEnStruct(nombreArchivo string) *globals.Metadata {
	// Paso 2: Abrir el archivo
	archivo, err := os.Open(nombreArchivo)
	if err != nil {
		return nil
	}
	defer archivo.Close()

	// Paso 3 y 4: Leer y deserializar el contenido del archivo en el struct
	bytes, err := io.ReadAll(archivo)
	if err != nil {
		return nil
	}

	var metadata globals.Metadata
	err = json.Unmarshal(bytes, &metadata)
	if err != nil {
		return nil
	}

	// El archivo se cierra automáticamente gracias a defer
	return &metadata
}

/**
  * ReadFs: Lee un archivo del sistema de archivos

  - @param nombreArchivo: nombre del archivo a leer
  - @param desplazamiento: desplazamiento en bytes desde el inicio del archivo
  - @param tamanio: cantidad de bytes a leer (si es -1, se lee todo el archivo)
  - @return contenido: contenido del archivo leído
*/
func ReadFs(nombreArchivo string, desplazamiento int, tamanio int) []byte {
	archivo := globals.Fcbs[nombreArchivo]
	var tamanioALeer int
	if tamanio == -1 {
		tamanioALeer = archivo.Size
	} else {
		tamanioALeer = tamanio
	}

	contenido := make([]byte, tamanioALeer)

	posBloqueInicial := archivo.InitialBlock - 1
	primerByteArchivo := posBloqueInicial*globals.ConfigIO.Dialfs_block_size + desplazamiento

	for i := 0; i < tamanioALeer; i++ {
		contenido[i] = globals.Blocks[primerByteArchivo+i]
	}

	return contenido
}

func WriteFs(contenido []byte, byteInicial int) {
	bloqueInicial := int(math.Max(1, math.Ceil(float64(byteInicial)/float64(globals.ConfigIO.Dialfs_block_size))))

	fmt.Println("WRITE - Byte inicial: ", byteInicial, "Bloque inicial: ", bloqueInicial)

	tamanioContenido := len(contenido)
	for i := 0; i < tamanioContenido; i++ {
		globals.Blocks[byteInicial+i] = contenido[i]
		fmt.Println("Byte: ", byteInicial+i, "Contenido: ", contenido[i])
	}

	tamanioFinalEnBloques := int(math.Ceil(float64(len(contenido)) / float64(globals.ConfigIO.Dialfs_block_size)))
	OcuparBloquesDesde(bloqueInicial, tamanioFinalEnBloques)
	ActualizarBloques()
}

func EntraEnDisco(tamanioTotalEnBloques int) int {
	for i := 0; i < globals.ConfigIO.Dialfs_block_count; i++ {
		espacioActual := CalcularBloquesLibreAPartirDe(i)

		if espacioActual >= tamanioTotalEnBloques {
			return i
		} else {
			i += espacioActual
		}
	}
	return (-1)
}

// * Manejo de BLOQUES
/**
 * ContadorDeEspaciosLibres: cuenta la cantidad de bloques libres TOTAL en el sistema de archivos
 */
func ContadorDeEspaciosLibres() int {
	var contador = 0
	for i := 0; i < globals.ConfigIO.Dialfs_block_count; i++ {
		if globals.CurrentBitMap[i] == 0 {
			contador++
		}
	}
	return contador
}

/**
 * CalcularBloquesLibreAPartirDe: calcula la cantidad de bloques libres a partir de la posición de un bloque inicial hasta encontrar uno seteado
 */
func CalcularBloquesLibreAPartirDe(posBloqueInicial int) int {
	var i = posBloqueInicial
	
	var contadorLibres = 0 // Inicializamos el contador de bloques libres

	for i < globals.ConfigIO.Dialfs_block_count {
		if IsNotSet(i) { // Si el bloque actual no está seteado (es 0),
			contadorLibres++ // Incrementamos el contador de bloques libres
		} else { // Si encontramos un bloque seteado (es 1),
			break // Terminamos la iteración
		}
		i++ // Pasamos al siguiente bloque
	}

	return contadorLibres // Devolvemos el contador de bloques libres
}

/**
 * CalcularBloqueLibre: calcula el primer bloque libre en el sistema de archivos
 */

func CalcularBloqueLibre() int {
	var i = 0
	for i < globals.ConfigIO.Dialfs_block_count {
		if IsNotSet(i) {
			break
		}
		i++
	}
	return i + 1
}

/**
 * LiberarBloquesDesde: libera bloques a partir de un bloque inicial hasta el tamaño a borrar
 */
func LiberarBloquesDesde(numBloque int, tamanioABorrar int) {
	var i = numBloque - 1
	var contador = 0
	for contador < tamanioABorrar {
		if i >= globals.ConfigIO.Dialfs_block_count {
			fmt.Println("Block index out of range: ", i)
			break
		}
		if !IsNotSet(i) {
			Clear(i)
			contador++
		} else {
			break
		}
		i++
	}
	ActualizarBitmap()
}

/**
 * LiberarBloque: libera bloques desde el bloque final del archivo hasta el tamaño a borrar
 */
func LiberarBloque(bloque int, tamanioABorrar int) {
	for i := 0; i < tamanioABorrar; i++ {
		Clear(bloque - i)
	}
	ActualizarBitmap()
}

/**
 * OcuparBloquesDesde: ocupa bloques a partir de un bloque inicial hasta el tamaño a setear
 */
func OcuparBloquesDesde(numBloque int, tamanioASetear int) {
	var i = numBloque - 1
	var contador = 0                // Inicializa el contador
	for contador < tamanioASetear { // Continúa mientras el contador sea menor que tamanioASetear
		if IsNotSet(i) { // Si el bloque actual no está seteado
			Set(i) // Setea el bloque
			fmt.Println("Se setteo el bloque ", i+1)
			contador++ // Incrementa el contador
		} else { // Si el byte ya está seteado
			break // Rompe el bucle
		}
		i++ // Incrementa el índice para revisar el siguiente bloque
	}
	ActualizarBitmap()
}

/**
 * ActualizarBloques: actualiza el archivo de bloques en el sistema de archivos
 */
func ActualizarBloques() {
	bloquesActualizado := globals.Blocks

	CrearModificarArchivo("dialfs/bloques.dat", bloquesActualizado)
}

// * Manejo de BITMAP
func NewBitMap(size int) []byte {
	NewBMAp := make([]byte, size)
	for i := 0; i < size; i++ {
		NewBMAp[i] = byte(0)
	}
	return NewBMAp
}

func Set(i int) {
	globals.CurrentBitMap[i] = byte(1)
}

func Clear(i int) {
	globals.CurrentBitMap[i] = byte(0)
}

func IsNotSet(i int) bool {
	return globals.CurrentBitMap[i] == byte(0)
}

func ActualizarBitmap() {
	bitmapActualizado := globals.CurrentBitMap

	CrearModificarArchivo("dialfs/bitmap.dat", bitmapActualizado)
}
