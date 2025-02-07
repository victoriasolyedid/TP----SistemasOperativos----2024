package clientutils

import (
	"bufio"
	"io"
	"log"
	"os"
)

/**
 * LeerConsola: Lee la consola hasta que se presione enter y guarda los mensajes en un archivo de log

 * @param logfile: Archivo de log
 */
func LeerConsola(logfile *os.File) {
	log.SetOutput(io.MultiWriter(os.Stdout, logfile))
	
	reader := bufio.NewReader(os.Stdin)
	log.Println("Ingrese los mensajes")

	i := 0
	for i < 1 {
		text, _ := reader.ReadString('\n')
		if text == "\n" {					// Normalmente se ejecuta en Linux, en windows el salto de linea es \r\n
			i++
		}

		_, err := logfile.WriteString(text)
		if err != nil {
			log.Fatal(err)
		}
	}
}