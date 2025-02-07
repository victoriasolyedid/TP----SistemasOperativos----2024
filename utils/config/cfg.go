package cfg

import (
	"encoding/json"
	"os"
	"strconv"
)

/*	ConfigInit:

	Hay que pasarle una interfaz como parámetro para que el decoder pueda
	decodificar el json con el formato de la interfaz especificada.
	Si no lo haces, decodifica con un map[string]interface{}.
*/
func ConfigInit(filePath string, config interface{}) error {
	configFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&config); err != nil {
		return err
	}

	return nil
}

func VEnvKernel(host *string, puerto *int) {
	if os.Getenv("KERNEL_HOST") != "" && host != nil {
		*host = os.Getenv("KERNEL_HOST")
	}
	if os.Getenv("KERNEL_PORT") != "" && puerto != nil {
		*puerto, _ = strconv.Atoi(os.Getenv("KERNEL_PORT"))
	}
}

func VEnvCpu(host *string, puerto *int) {
	if os.Getenv("CPU_HOST") != "" && host != nil {
		*host = os.Getenv("CPU_HOST")
	}
	if os.Getenv("CPU_PORT") != "" && puerto != nil {
		*puerto, _ = strconv.Atoi(os.Getenv("CPU_PORT"))
	}
}

func VEnvMemoria(host *string, puerto *int) {
	if os.Getenv("MEMORIA_HOST") != "" && host != nil {
		*host = os.Getenv("MEMORIA_HOST")
	}
	if os.Getenv("MEMORIA_PORT") != "" && puerto != nil {
		*puerto, _ = strconv.Atoi(os.Getenv("MEMORIA_PORT"))
	}
}

func VEnvIO(host *string, puerto *int) {
	if os.Getenv("IO_HOST") != "" && host != nil {
		*host = os.Getenv("IO_HOST")
	}
	if os.Getenv("IO_PORT") != "" && puerto != nil {
		*puerto, _ = strconv.Atoi(os.Getenv("IO_PORT"))
	}
}