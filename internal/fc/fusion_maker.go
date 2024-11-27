package fc

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/grussorusso/serverledge/internal/function"
)

// CombineFunctions fuse the code of two functions
func CombineFunctions(fun1, fun2 *function.Function) (*function.Function, error) {
	// Decode both functions' tar codes
	tarFun1, err := decodeBase64Tar(fun1.TarFunctionCode)
	if err != nil {
		return nil, fmt.Errorf("error in fun1 decoding: %w", err)
	}

	tarFun2, err := decodeBase64Tar(fun2.TarFunctionCode)
	if err != nil {
		return nil, fmt.Errorf("error in fun2 decoding: %w", err)
	}

	// Combine the tar files
	combinedTar, err := combineTarFiles(tarFun1, tarFun2, fun1.Handler, fun2.Handler, fun1.Name, fun2.Name, fun1.Signature, fun2.Signature)
	if err != nil {
		return nil, fmt.Errorf("error in Tar files combination: %w", err)
	}

	//ONLY FOR DEBUG
	err = saveTarToFile(combinedTar, "../DebugTar/combined_function.tar")
	if err != nil {
		fmt.Printf("Error in Saving Tar file : %v\n", err)
	}

	// Combined Function Creation, still to tune hyperparameters
	combinedFunction := &function.Function{
		Name:            fun1.Name + "_" + fun2.Name,
		Runtime:         fun1.Runtime,
		MemoryMB:        fun1.MemoryMB + fun2.MemoryMB,
		CPUDemand:       fun1.CPUDemand + fun2.CPUDemand,
		Handler:         "combined_handler.central_handler",
		CustomImage:     fun1.CustomImage,
		Signature:       combineSignatures(fun1.Signature, fun2.Signature),
		TarFunctionCode: base64.StdEncoding.EncodeToString(combinedTar),
	}

	return combinedFunction, nil
}

// decodeBase64Tar decodes a base64 string in a tar archive
func decodeBase64Tar(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// combineTarFiles combines two tar archives, handled by the brand new file combined_handler.py
// Homonymy is handled with dynamic renaming extended to the imports for each file in each package
func combineTarFiles(tar1, tar2 []byte, handler1, handler2, NameFun1, NameFun2 string, sig1, sig2 *function.Signature) ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	//Adding of the files of the first archive with handling of homonymy
	//and relative imports using function name as a prefix
	if err := addTarContentsWithUpdatedImports(tw, tar1, NameFun1); err != nil {
		return nil, err
	}

	//Adding of the files of the second archive with handling of homonymy
	//and relative imports using function name as a prefix
	if err := addTarContentsWithUpdatedImports(tw, tar2, NameFun2); err != nil {
		return nil, err
	}

	// Generation of combined_handler's code
	combinedCode, err := generateCombinedHandlerWithNamespaces(NameFun1, handler1, NameFun2, handler2, sig1, sig2)
	if err != nil {
		return nil, err
	}

	// Added
	if err := addFileToTar(tw, "combined_handler.py", combinedCode); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// This functions adds files of a single archive with handling of homonymy and relative imports using function name as a prefix
func addTarContentsWithUpdatedImports(tw *tar.Writer, tarContents []byte, prefix string) error {
	tr := tar.NewReader(bytes.NewReader(tarContents))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		var fileContents bytes.Buffer
		if _, err := io.Copy(&fileContents, tr); err != nil {
			return err
		}

		// Check if it's a Python File
		if strings.HasSuffix(header.Name, ".py") {
			updatedCode, err := updatePythonImports(fileContents.String(), prefix)
			if err != nil {
				return err
			}
			fileContents = *bytes.NewBufferString(updatedCode)
		}

		// Modifying file name with prefix used as a directory
		header.Name = fmt.Sprintf("%s/%s", prefix, header.Name)

		// Header update
		header.Size = int64(fileContents.Len())
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(tw, &fileContents); err != nil {
			return err
		}
	}
	return nil
}

// Updates of the imports with the new prefix necessary to handling directory creation
func updatePythonImports(originalCode, prefix string) (string, error) {
	// Regex to find all import entries in the code
	importPattern := `(?m)^(from|import)\s+([a-zA-Z_][a-zA-Z0-9_]*)(\s+import\s+.*)?`
	re := regexp.MustCompile(importPattern)

	// Replacing with updated import definition
	updatedCode := re.ReplaceAllStringFunc(originalCode, func(match string) string {
		parts := strings.Fields(match)
		if len(parts) < 2 {
			// Wrong format
			return match
		}

		// Modifica il modulo importato aggiungendo il prefisso e mantenendo i nomi originali
		//Modifying the imported module adding prefix and maintaining the original names to maintain unchanged the rest of the code
		if parts[0] == "import" {
			// handling of multiple imports
			modules := strings.Split(parts[1], ",")
			for i, mod := range modules {
				modules[i] = fmt.Sprintf("%s.%s as %s", prefix, strings.TrimSpace(mod), strings.TrimSpace(mod))
			}
			return fmt.Sprintf("import %s", strings.Join(modules, ", "))
		} else if parts[0] == "from" && len(parts) >= 3 {
			// Handling of multiple imports from a specific module
			importedFunctions := strings.Split(parts[2], ",")
			for i, funcName := range importedFunctions {
				importedFunctions[i] = fmt.Sprintf("%s as %s", strings.TrimSpace(funcName), strings.TrimSpace(funcName)) // aggiunge "as" mantenendo il nome originale
			}
			return fmt.Sprintf("from %s.%s import %s", prefix, parts[1], strings.Join(importedFunctions, ", "))
		}
		return match

	})

	return updatedCode, nil
}

func generateCombinedHandlerWithNamespaces(prefix1, handler1, prefix2, handler2 string, sig1, sig2 *function.Signature) (string, error) {
	// Parsing handler, mod1 the module (es. inc) and func1 is the defined function handler
	mod1, func1, err := parseHandler(handler1)
	if err != nil {
		return "", err
	}
	mod2, func2, err := parseHandler(handler2)
	if err != nil {
		return "", err
	}

	//Signature handling to correcly handle the params passing
	mappingLogic, err := generateMappingLogic(sig1, sig2)
	if err != nil {
		return "", err
	}

	// Combined handler.py final code generated as follows
	code := fmt.Sprintf(`
from %s.%s import %s as %s_%s
from %s.%s import %s as %s_%s

def central_handler(params, context):
    # Esegui il primo handler
    intermediate_result = %s_%s(params, context)

    # Adatta gli output del primo handler come input per il secondo
%s
    # Esegui il secondo handler
    final_result = %s_%s(transformed_params, context)
    return final_result
`, prefix1, mod1, func1, prefix1, func1, prefix2, mod2, func2, prefix2, func2, prefix1, func1, mappingLogic, prefix2, func2)

	return code, nil
}

// It extracts the module and the defined function handler from the handler definition
func parseHandler(handler string) (module, function string, err error) {
	parts := bytes.Split([]byte(handler), []byte("."))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid handler: %s", handler)
	}
	return string(parts[0]), string(parts[1]), nil
}

func generateMappingLogic(sig1, sig2 *function.Signature) (string, error) {
	if len(sig1.Outputs) != len(sig2.Inputs) {
		return "", fmt.Errorf("Number of output and input are not corresponding")
	}

	logic := "    transformed_params = {\n"
	for i, _ := range sig1.Outputs {
		input := sig2.Inputs[i]
		logic += fmt.Sprintf("        '%s': intermediate_result,\n", input.Name)
	}
	logic += "    }\n"
	return logic, nil
}

func addFileToTar(tw *tar.Writer, filename, content string) error {
	header := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return err
	}
	return nil
}

// combineSignatures combines two functions' signatures
func combineSignatures(sig1, sig2 *function.Signature) *function.Signature {
	return &function.Signature{
		Inputs:  sig1.Inputs,
		Outputs: sig2.Outputs,
	}
}

// ONLY FOR DEBUG
func saveTarToFile(tarData []byte, filePath string) error {
	// Create or overwrite the tar file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error in file creation: %w", err)
	}
	defer file.Close()

	_, err = file.Write(tarData)
	if err != nil {
		return fmt.Errorf("error writing the file: %w", err)
	}

	fmt.Printf("File TAR saved with success in %s\n", filePath)
	return nil
}
