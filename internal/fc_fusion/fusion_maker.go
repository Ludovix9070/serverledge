package fc_fusion

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
		//Name:            fun1.Name + "_" + fun2.Name,
		Name:            GenerateFunctionHash(fun1.Name, fun2.Name),
		Runtime:         fun1.Runtime,
		MemoryMB:        MaxInt64(fun1.MemoryMB, fun2.MemoryMB),
		CPUDemand:       MaxFloat64(fun1.CPUDemand, fun2.CPUDemand),
		Handler:         "mypkg.combined_handler.central_handler",
		CustomImage:     fun1.CustomImage,
		Signature:       combineSignatures(fun1.Signature, fun2.Signature),
		TarFunctionCode: base64.StdEncoding.EncodeToString(combinedTar),
	}

	err = saveCombinedFunction(combinedFunction)
	if err != nil {
		return nil, err
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
	/*if err := addTarContentsWithUpdatedImports(tw, tar1, NameFun1); err != nil {
		return nil, err
	}*/

	if err := addTarContentsWithUpdatedImports(tw, tar1, "mypkg/A"); err != nil {
		return nil, err
	}

	//Adding of the files of the second archive with handling of homonymy
	//and relative imports using function name as a prefix
	/*if err := addTarContentsWithUpdatedImports(tw, tar2, NameFun2); err != nil {
		return nil, err
	}*/

	if err := addTarContentsWithUpdatedImports(tw, tar2, "mypkg/B"); err != nil {
		return nil, err
	}

	// Generation of combined_handler's code
	/*combinedCode, err := generateCombinedHandlerWithNamespaces(NameFun1, handler1, NameFun2, handler2, sig1, sig2)
	if err != nil {
		return nil, err
	}*/

	// Generation of combined_handler's code
	combinedCode, err := generateCombinedHandlerWithNamespaces("A", handler1, "B", handler2, sig1, sig2)
	if err != nil {
		return nil, err
	}

	// Added
	if err := addFileToTar(tw, "mypkg/combined_handler.py", combinedCode); err != nil {
		return nil, err
	}

	/*if err := addFileToTar(tw, "mypkg/__init__.py", ""); err != nil {
		return nil, fmt.Errorf("errore nell'aggiunta di __init__.py: %w", err)
	}*/

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

/* // funziona ma non gestisci differenti mypkg
// This functions adds files of a single archive with handling of homonymy and relative imports using function name as a prefix
func addTarContentsWithUpdatedImports(tw *tar.Writer, tarContents []byte, prefix string) error {
	shouldUpdateImports, err := shouldUpdatePythonImports(tarContents)
	if err != nil {
		return err
	}

	tr := tar.NewReader(bytes.NewReader(tarContents))

	seenDirectories := make(map[string]bool)   // Keep track of directories already processed
	existingInitFiles := make(map[string]bool) // Track directories with __init__.py

	// First pass: collect existing __init__.py files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Check if the file is an __init__.py
		if strings.HasSuffix(header.Name, "__init__.py") {
			dir := getDirectoryFromPath(header.Name)
			existingInitFiles[dir] = true
		}
	}

	// Reset tar reader for the second pass
	tr = tar.NewReader(bytes.NewReader(tarContents))

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		//OPTIONAL->add __init__.py
		// Capture directory path for the file
		dir := fmt.Sprintf("%s/%s", prefix, getDirectoryFromPath(header.Name))

		// Add an __init__.py file for this directory if not already present
		if dir != "" && !seenDirectories[dir] {
			if !existingInitFiles[getDirectoryFromPath(header.Name)] {
				if err := addInitFileToTar(tw, dir, seenDirectories); err != nil {
					return err
				}
			}
			seenDirectories[dir] = true
		}

		var fileContents bytes.Buffer
		if _, err := io.Copy(&fileContents, tr); err != nil {
			return err
		}

		// Check if it's a Python File
		if strings.HasSuffix(header.Name, ".py") && shouldUpdateImports {
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
}*/

/*ok con relativi ma duplicati init
func addTarContentsWithUpdatedImports(tw *tar.Writer, tarContents []byte, prefix string) error {
	shouldUpdateImports, err := shouldUpdatePythonImports(tarContents)
	if err != nil {
		return err
	}

	tr := tar.NewReader(bytes.NewReader(tarContents))

	seenDirectories := make(map[string]bool)   // Keep track of directories already processed
	existingInitFiles := make(map[string]bool) // Track directories with __init__.py

	// First pass: collect existing __init__.py files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Check if the file is an __init__.py
		if strings.HasSuffix(header.Name, "__init__.py") {
			dir := getDirectoryFromPath(header.Name)
			existingInitFiles[dir] = true
		}
	}

	// Reset tar reader for the second pass
	tr = tar.NewReader(bytes.NewReader(tarContents))

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Read file contents
		var fileContents bytes.Buffer
		if _, err := io.Copy(&fileContents, tr); err != nil {
			return err
		}

		// Verifica se il percorso contiene già una sottostruttura `mypkg`
		parts := strings.Split(header.Name, "/")
		newPath := header.Name

		// Se il percorso contiene già `mypkg`, sostituisci solo la prima parte
		if len(parts) > 1 && parts[0] == "mypkg" {
			parts[0] = prefix
			newPath = strings.Join(parts, "/")
		} else {
			// Altrimenti, aggiungi il prefisso una sola volta
			newPath = fmt.Sprintf("%s/%s", prefix, header.Name)
		}

		// Rimuovi eventuali slash finali
		newPath = strings.TrimSuffix(newPath, "/")

		// Aggiorna header.Name
		header.Name = newPath

		// Aggiorna il contenuto del file Python se necessario
		if strings.HasSuffix(header.Name, ".py") && shouldUpdateImports {
			updatedCode, err := updatePythonImports(fileContents.String(), prefix)
			if err != nil {
				return err
			}
			fileContents = *bytes.NewBufferString(updatedCode)
		}

		// Aggiungi __init__.py alla directory, se necessario
		dir := getDirectoryFromPath(newPath)
		if dir != "" && !seenDirectories[dir] {
			// Aggiungi __init__.py solo se non già presente
			if !existingInitFiles[dir] {
				if err := addInitFileToTar(tw, dir, seenDirectories); err != nil {
					return err
				}
			}
			seenDirectories[dir] = true
		}

		// Aggiorna l'header con la nuova dimensione del file
		header.Size = int64(fileContents.Len())
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(tw, &fileContents); err != nil {
			return err
		}
	}

	return nil
}*/

/* //versione ok ma init sbagliati
func addTarContentsWithUpdatedImports(tw *tar.Writer, tarContents []byte, prefix string) error {
	shouldUpdateImports, err := shouldUpdatePythonImports(tarContents)
	if err != nil {
		return err
	}

	tr := tar.NewReader(bytes.NewReader(tarContents))

	seenDirectories := make(map[string]bool)   // Keep track of directories already processed
	existingInitFiles := make(map[string]bool) // Track directories with __init__.py

	// First pass: collect existing __init__.py files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Check if the file is an __init__.py and save its full directory path
		if strings.HasSuffix(header.Name, "__init__.py") {
			dir := getDirectoryFromPath(header.Name)
			fullDirPath := fmt.Sprintf("%s/%s", prefix, dir)
			existingInitFiles[fullDirPath] = true
		}
	}

	// Reset tar reader for the second pass
	tr = tar.NewReader(bytes.NewReader(tarContents))

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Read file contents
		var fileContents bytes.Buffer
		if _, err := io.Copy(&fileContents, tr); err != nil {
			return err
		}

		// Verifica se il percorso contiene già una sottostruttura `mypkg`
		parts := strings.Split(header.Name, "/")
		newPath := header.Name
		original_dir := getDirectoryFromPath(header.Name)

		// Se il percorso contiene già `mypkg`, sostituisci solo la prima parte
		if len(parts) > 1 && parts[0] == "mypkg" {
			parts[0] = prefix
			newPath = strings.Join(parts, "/")
		} else {
			// Altrimenti, aggiungi il prefisso una sola volta
			newPath = fmt.Sprintf("%s/%s", prefix, header.Name)
		}

		// Rimuovi eventuali slash finali
		newPath = strings.TrimSuffix(newPath, "/")

		// Aggiorna header.Name
		header.Name = newPath

		// Aggiorna il contenuto del file Python se necessario
		if strings.HasSuffix(header.Name, ".py") && shouldUpdateImports {
			updatedCode, err := updatePythonImports(fileContents.String(), prefix)
			if err != nil {
				return err
			}
			fileContents = *bytes.NewBufferString(updatedCode)
		}

		// Aggiungi __init__.py alla directory, se necessario
		dir := getDirectoryFromPath(newPath)
		if dir != "" && !seenDirectories[dir] {
			if original_dir != "" && !seenDirectories[original_dir] {
				// Aggiungi __init__.py solo se non già presente
				if !existingInitFiles[dir] {
					if !existingInitFiles[original_dir] {
						if err := addInitFileToTar(tw, dir, seenDirectories); err != nil {
							return err
						}
						existingInitFiles[dir] = true // Aggiorna che ora il file è stato aggiunto
					}
				}
				seenDirectories[dir] = true
			}
		}

		// Aggiorna l'header con la nuova dimensione del file
		header.Size = int64(fileContents.Len())
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(tw, &fileContents); err != nil {
			return err
		}
	}

	return nil
}*/

func addTarContentsWithUpdatedImports(tw *tar.Writer, tarContents []byte, prefix string) error {
	shouldUpdateImports, err := shouldUpdatePythonImports(tarContents)
	if err != nil {
		return err
	}

	tr := tar.NewReader(bytes.NewReader(tarContents))

	//seenDirectories := make(map[string]bool)   // Keep track of directories already processed
	//existingInitFiles := make(map[string]bool) // Track directories with __init__.py

	/*// First pass: collect existing __init__.py files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Check if the file is an __init__.py and save its full directory path
		if strings.HasSuffix(header.Name, "__init__.py") {
			dir := getDirectoryFromPath(header.Name)
			fullDirPath := fmt.Sprintf("%s/%s", prefix, dir)
			existingInitFiles[fullDirPath] = true
		}
	}

	// Reset tar reader for the second pass
	tr = tar.NewReader(bytes.NewReader(tarContents))*/

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Read file contents
		var fileContents bytes.Buffer
		if _, err := io.Copy(&fileContents, tr); err != nil {
			return err
		}

		// Determine the new path and original directory
		parts := strings.Split(header.Name, "/")
		newPath := header.Name

		// Check if the path already contains "mypkg"
		if len(parts) > 1 && parts[0] == "mypkg" {
			parts[0] = prefix
			newPath = strings.Join(parts, "/")
		} else {
			// Otherwise, add the prefix
			newPath = fmt.Sprintf("%s/%s", prefix, header.Name)
		}

		// Remove trailing slashes
		newPath = strings.TrimSuffix(newPath, "/")

		// Update header.Name
		header.Name = newPath

		// Update Python imports if necessary
		if strings.HasSuffix(header.Name, ".py") && shouldUpdateImports {
			updatedCode, err := updatePythonImports(fileContents.String(), prefix)
			if err != nil {
				return err
			}
			fileContents = *bytes.NewBufferString(updatedCode)
		}

		/*// Ensure __init__.py exists in the directory
		dir := getDirectoryFromPath(newPath)

		// Add __init__.py only if it's not already added in this directory
		if dir != "" && !seenDirectories[dir] {
			// Skip adding __init__.py if it already exists in this directory
			if !existingInitFiles[dir] {
				// Ensure __init__.py is added to this directory
				if err := addInitFileToTar(tw, dir, seenDirectories); err != nil {
					return err
				}
				// Mark this directory as having an __init__.py file
				existingInitFiles[dir] = true
			}
			// Mark the directory as processed
			seenDirectories[dir] = true
		}*/

		// Update the header with the new file size
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

// Check if it's necessary to update imports-->only if the tar contains only files
func shouldUpdatePythonImports(tarContents []byte) (bool, error) {
	tr := tar.NewReader(bytes.NewReader(tarContents))
	hasOnlyFiles := true

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}

		// ONLY FOR DEBUG
		//fmt.Printf("Processing file: %s, Typeflag: %d\n", header.Name, header.Typeflag)

		// Extract dir from path
		dir := getDirectoryFromPath(header.Name)

		if header.Typeflag == tar.TypeDir || dir != "" {
			//fmt.Printf("Detected directory: %s\n", dir)
			hasOnlyFiles = false
			break
		}
	}

	return hasOnlyFiles, nil
}

// Helper function to extract directory path from a file path
func getDirectoryFromPath(filePath string) string {
	if idx := strings.LastIndex(filePath, "/"); idx != -1 {
		return filePath[:idx]
	}
	return ""
}

func addInitFileToTar(tw *tar.Writer, dir string, seenDirectories map[string]bool) error {
	// Controlla se il file è già stato aggiunto
	if seenDirectories[dir] {
		return nil
	}

	// Aggiungi il file `__init__.py` alla directory
	filename := fmt.Sprintf("%s/__init__.py", strings.TrimSuffix(dir, "/"))
	header := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: 0, // File vuoto
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// Segna la directory come già processata
	seenDirectories[dir] = true
	return nil
}

/* //versione funzionante prima degli import relativi
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
*/

// Updates of the imports with the new prefix necessary to handling directory creation
/*func updatePythonImports(originalCode, prefix string) (string, error) {
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

		//Modifying the imported module adding prefix and maintaining the original names to maintain unchanged the rest of the code
		if parts[0] == "import" {
			// handling of multiple imports
			modules := strings.Split(parts[1], ",")
			for i, mod := range modules {
				modules[i] = fmt.Sprintf("%s", strings.TrimSpace(mod))
			}
			return fmt.Sprintf("from . import %s", strings.Join(modules, ", "))
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
}*/
/*//forse meglio per import relativi, da testare
func updatePythonImports(originalCode, prefix string) (string, error) {
    importPattern := `(?m)^(from\s+|import\s+)([a-zA-Z_][a-zA-Z0-9_]*)(.*)?$`
    re := regexp.MustCompile(importPattern)

    updatedCode := re.ReplaceAllStringFunc(originalCode, func(match string) string {
        parts := strings.Fields(match)
        if len(parts) < 2 {
            return match
        }
        if strings.HasPrefix(match, "from") {
            return fmt.Sprintf("from .%s.%s%s", prefix, parts[1], strings.Join(parts[2:], " "))
        } else if strings.HasPrefix(match, "import") {
            return fmt.Sprintf("from .%s import %s", prefix, parts[1])
        }
        return match
    })

    return updatedCode, nil
}
*/

func updatePythonImports(originalCode, prefix string) (string, error) {
	// Codice da aggiungere
	/*setupCode := `import sys, os
	def find_and_add_to_sys_path():
	    # Usa il percorso assoluto dello script corrente
	    start_dir = os.path.dirname(os.path.abspath(__file__))
	    print("sono in normal handler")
		sys.path.insert(0, start_dir)

	find_and_add_to_sys_path()
	`

		// Controlla se il codice per find_and_add_to_sys_path() è già presente
		if !strings.Contains(originalCode, "find_and_add_to_sys_path()") {
			// Inserisce il codice all'inizio del file
			originalCode = setupCode + "\n" + originalCode
		}*/

	// Regex per trovare tutti gli import nel codice
	importPattern := `(?m)^(from|import)\s+([a-zA-Z_][a-zA-Z0-9_]*)(\s+import\s+.*)?`
	re := regexp.MustCompile(importPattern)

	// Sostituzione degli import
	updatedCode := re.ReplaceAllStringFunc(originalCode, func(match string) string {
		parts := strings.Fields(match)
		if len(parts) < 2 {
			// Formato non valido
			return match
		}

		// Modifica del modulo importato aggiungendo il prefisso
		if parts[0] == "import" {
			// Gestione di più moduli importati
			modules := strings.Split(parts[1], ",")
			for i, mod := range modules {
				modules[i] = fmt.Sprintf("%s", strings.TrimSpace(mod))
			}
			return fmt.Sprintf("from . import %s", strings.Join(modules, ", "))
		} else if parts[0] == "from" && len(parts) >= 3 {
			// Gestione di più elementi importati da un modulo specifico
			importedFunctions := strings.Split(parts[2], ",")
			for i, funcName := range importedFunctions {
				importedFunctions[i] = fmt.Sprintf("%s as %s", strings.TrimSpace(funcName), strings.TrimSpace(funcName)) // Aggiunge "as" mantenendo il nome originale
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

import sys, os

def restrict_import_to_current_dir():
    current_dir = os.path.dirname(os.path.abspath(__file__))
    sys.path = [current_dir, os.path.join(current_dir, "%s"), os.path.join(current_dir, "%s")]

restrict_import_to_current_dir()

from .%s.%s import %s as %s_%s
from .%s.%s import %s as %s_%s

def central_handler(params, context):
    # Esegui il primo handler
    intermediate_result = %s_%s(params, context)

    # Adatta gli output del primo handler come input per il secondo
%s
    # Esegui il secondo handler
    final_result = %s_%s(transformed_params, context)
    return final_result
`, prefix1, prefix2, prefix1, mod1, func1, prefix1, func1, prefix2, mod2, func2, prefix2, func2, prefix1, func1, mappingLogic, prefix2, func2)

	return code, nil
}

/* //versione prima degli import relativi
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

import sys, os

def find_and_add_to_sys_path():
    start_dir = os.path.dirname(os.path.abspath(__file__))
    sys.path.insert(0, start_dir)

find_and_add_to_sys_path()

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
}*/

// It extracts the module and the defined function handler from the handler definition
/*func parseHandler(handler string) (module, function string, err error) {
	parts := bytes.Split([]byte(handler), []byte("."))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid handler: %s", handler)
	}
	return string(parts[0]), string(parts[1]), nil
}*/

func parseHandler(handler string) (module, function string, err error) {
	parts := bytes.Split([]byte(handler), []byte("."))
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid handler: %s", handler)
	}
	return string(parts[len(parts)-2]), string(parts[len(parts)-1]), nil
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

func saveCombinedFunction(f *function.Function) error {
	_, ok := function.GetFunction(f.Name)
	if ok {
		//return fmt.Errorf("dropping fusionized function for already existing function '%s'\n", f.Name)
		return nil
	}

	err := f.SaveToEtcd()
	if err != nil {
		return fmt.Errorf("error in fusionized function creation: %w", err)
	}

	return nil
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func MaxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// GenerateFunctionHash genera un hash SHA256 basato sui nomi delle due funzioni
func GenerateFunctionHash(name1, name2 string) string {
	hasher := sha256.New()
	hasher.Write([]byte(name1 + name2))
	return hex.EncodeToString(hasher.Sum(nil)) // Restituisce l'hash completo (64 caratteri)
}
