package logger

import (
	"io"
	"log"
	"os"
)

/**
 * LogfileCreate: Crea un archivo de log
 * Debe ejecutarse solo una vez al inicio de cada módulo. Cada vez que se ejecuta se sobreescribe el archivo.
 * Únicamente se escribe lo que se elija

 * @param filepath: Ruta del archivo
 * @return *os.File: Archivo de log
 */
func LogfileCreate(filepath string) (*os.File, error) {
	logfile, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logfile.Close()

	return logfile, nil
}

/**
 * ConfigurarLogger: Configura un logger general para todo el trabajo
*/
func ConfigurarLogger(path string) {
	logFile, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
