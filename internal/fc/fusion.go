package fc

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	//"compress/gzip"
	//"encoding/json"
	//"sync"
	//"time"
	//"github.com/grussorusso/serverledge/internal/function"
	//"github.com/grussorusso/serverledge/internal/node"
	//"github.com/grussorusso/serverledge/internal/scheduling"
	//"github.com/grussorusso/serverledge/internal/types"
	//"github.com/labstack/gommon/log"
	//"github.com/lithammer/shortuuid"
)

func decodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

func extractTar(tarData []byte) (string, error) {
	tarBytes, err := decodeBase64(string(tarData))
	if err != nil {
		fmt.Println("Errore nella decodifica Base64:", err)
		return "", fmt.Errorf("the function cannot be converted from base64")
	}
	buf := new(strings.Builder)
	tarReader := tar.NewReader(bytes.NewReader(tarBytes))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			fmt.Println("Errore durante la lettura:", err)
			return "", fmt.Errorf("the function cannot be extracted")
		}

		if header.Typeflag == tar.TypeReg {
			//fmt.Printf("File: %s\n", header.Name)
			//buf := new(strings.Builder)
			io.Copy(buf, tarReader)
			//fmt.Printf("Contenuto:\n%s\n", buf.String())
		}
	}

	return buf.String(), nil
}

func extractFunctionName(code string) (string, error) {
	re := regexp.MustCompile(`(?m)^def\s+([a-zA-Z_]\w*)\s*\(`)
	match := re.FindStringSubmatch(code)
	if len(match) < 2 {
		return "", fmt.Errorf("Nome della funzione non trovato")
	}
	return match[1], nil
}

// Funzione per sostituire il nome della funzione con un nuovo nome
func renameFunction(code, originalName, newName string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^def\s+%s\s*\(`, originalName))
	return re.ReplaceAllString(code, fmt.Sprintf("def %s(", newName))
}

func makeFusion(firstFunc string, secondFunc string) (string, error) {
	//return base64.StdEncoding.DecodeString(data)
	funcName1, err := extractFunctionName(firstFunc)
	if err != nil {
		fmt.Println("Errore nell'estrazione del nome della prima funzione:", err)
		return "", fmt.Errorf("the first function name cannot be extracted")
	}
	funcName2, err := extractFunctionName(secondFunc)
	if err != nil {
		fmt.Println("Errore nell'estrazione del nome della seconda funzione:", err)
		return "", fmt.Errorf("the second function name cannot be extracted")
	}

	counter := 1
	renamedFunc1, renamedFunc2 := firstFunc, secondFunc

	// Verifica se i nomi delle funzioni sono uguali
	if funcName1 == funcName2 {
		// Rinomina entrambe le funzioni in modo univoco
		newFuncName1 := funcName1 + strconv.Itoa(counter)
		counter++
		newFuncName2 := funcName2 + strconv.Itoa(counter)

		renamedFunc1 = renameFunction(firstFunc, funcName1, newFuncName1)
		renamedFunc2 = renameFunction(secondFunc, funcName2, newFuncName2)

		funcName1, funcName2 = newFuncName1, newFuncName2
	}

	finalCode := fmt.Sprintf(`
%s

%s

def combined_handler(initial_params, context):
    # Esegui la prima funzione
    intermediate_result = %s(initial_params, context)

    # Prepara i parametri per la seconda funzione
    params_for_second = {"input": intermediate_result}

    # Esegui la seconda funzione e restituisci il risultato
    final_result = %s(params_for_second, context)
    return final_result
`, renamedFunc1, renamedFunc2, funcName1, funcName2)

	// Stampa il codice finale per verifica
	//fmt.Println("Codice unito:")
	//fmt.Println(finalCode)

	return finalCode, nil

}
