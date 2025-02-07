package generics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

/**
 * Hace un request HTTP y deserializa el body de la respuesta en el objeto que se pasa por parametro

 * @param method: metodo HTTP a usar
 * @param url: url a la que se hace el request
 * @param requestBody: cuerpo del request
 * @param responseBody: objeto en el que se deserializa el body de la respuesta
 * @return error: error en caso de que haya ocurrido alguno
 */
func DoRequest(method, url string, requestBody interface{}, responseBody interface{}) error {
	var reqBody []byte
	var err error
	if requestBody != nil {
		reqBody, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("error al serializar el body: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error al crear el request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error al hacer el request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("no se recibio una respuesta del tipo 2xx: %d %s", resp.StatusCode, resp.Status)
	}

	if responseBody != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error al leer el body: %v", err)
		}
		err = json.Unmarshal(body, responseBody)
		if err != nil {
			return fmt.Errorf("error al deserializar el body: %v", err)
		}
	}

	return nil
}