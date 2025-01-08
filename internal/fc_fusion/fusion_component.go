package fc_fusion

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
)

var metricInfos chan *returnedOutputData

// var fusionInvocationChannel chan *fc.FunctionComposition
var fusionInvocationChannel chan *fusionRequest
var dataMap map[time.Time]returnedOutputData

func Run(p FusionPolicy) {
	metricInfos = make(chan *returnedOutputData, 500)
	fusionInvocationChannel = make(chan *fusionRequest, 500)
	dataMap = make(map[time.Time]returnedOutputData)

	// initialize Resources
	//forse info saranno utili nel corso della vita del componente di fusione
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current resources for fusion: %v\n", &node.Resources)

	// initialize fusion policy
	p.Init()
	log.Println("Fusion Component started.")

	//var r *ReturnedOutputData
	var f *fusionRequest
	for {
		select {
		/*case r = <-metricInfos: // receive composition infos
		go p.OnArrival(r, nil)*/

		case f = <-fusionInvocationChannel: // receive composition infos
			go p.OnArrival(f)
		}

	}

}

func SubmitInfos(data returnedOutputData) error {
	metricInfos <- &data // send infos

	return nil
}

func SubmitFusionRequest(fc *fc.FunctionComposition) error {
	QueryStarter <- fc.Name // send fc name to get metrics from Prometheus
	//wait for the metrics data
	r := <-metricInfos
	saveInfos(*r)
	//fusionDecide() //da rimuovere, solo per vedere se stampa bene
	//QUI DEVO USARE LE METRICHE PER VALUTARE IL DAG

	fusionRequest := fusionRequest{
		composition:   fc,
		returnChannel: make(chan fusionResult, 1)}
	fusionInvocationChannel <- &fusionRequest // send request

	fusionResult, ok := <-fusionRequest.returnChannel
	if !ok {
		return fmt.Errorf("could not schedule the request")
	}

	fmt.Println("Fusion Result ", fusionResult.action)

	return nil
}

func fusionDecide() {
	//saveInfos
	//for all the fc? TODO
	//dataMap[infos.Timestamp] = infos
	//Dummy
	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		for key := range dataMap {
			fmt.Println("------------------------------------------")
			fmt.Println("Timestamp Key: ", key)
			fmt.Println("Metrics: ", dataMap[key])
			fmt.Println("------------------------------------------")
		}
		fmt.Println("")
	} else {
		fmt.Println("DON'T FUSE HERE with total dataMap: ", dataMap)
	}
}

func fusionSingleFcDecide(fr *fusionRequest) {
	//saveInfos
	//for all the fc? TODO
	//dataMap[infos.Timestamp] = infos
	//Dummy
	fmt.Println("FUSE COMMAND with Default/Alwaysfuse policy for composition ", fr.composition.Name)
	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		for key := range dataMap {
			fmt.Println("------------------------------------------")
			fmt.Println("Timestamp Key: ", key)
			fmt.Println("Metrics: ", dataMap[key].returnedInfos)
			fmt.Println("------------------------------------------")
		}
		fmt.Println("")

		ok, _ := FuseFc(fr.composition)
		if !ok {
			fr.returnChannel <- fusionResult{action: NOOP}
		} else {
			fmt.Println("new functions vector is ", fr.composition.Functions)
			fr.returnChannel <- fusionResult{action: FUSED}
		}
	} else {
		fmt.Println("DON'T FUSE HERE with total dataMap: ", dataMap)
		fr.returnChannel <- fusionResult{action: NOOP}
	}

}

func fusionEvaluate(fr *fusionRequest, policyDef policyDefinition) {
	//saveInfos
	//for all the fc? TODO
	//dataMap[infos.Timestamp] = infos
	//Dummy
	fmt.Println("FUSE COMMAND with Evaluate Policy for composition ", fr.composition.Name)
	for key := range dataMap {
		fmt.Println("------------------------------------------")
		fmt.Println("Timestamp Key: ", key)
		fmt.Println("Metrics: ", dataMap[key].returnedInfos)
		fmt.Println("------------------------------------------")
	}
	fmt.Println("")

	var latestTime time.Time
	var latestData returnedOutputData

	for timestamp, data := range dataMap {
		if timestamp.After(latestTime) {
			latestTime = timestamp
			latestData = data
			fmt.Println(latestData)
		}
	}

	otherWorkFunc, error := retrieveWorkflowsFunctions(fr.composition.Name)
	if error != nil {
		log.Println(error)
		fr.returnChannel <- fusionResult{action: NOOP}
	}

	log.Println(otherWorkFunc)

	/*for key := range fr.Composition.Functions {
		fmt.Println("------------------------------------------")
		fmt.Println("Function: ", fr.Composition.Functions[key].Name)
		fmt.Println("------------------------------------------")

		if contains(otherWorkFunc, fr.Composition.Functions[key].Name) {
			//da non fondere, ma per ora la fondo
			fmt.Printf("Funzione %s già usata in un altro workflow", fr.Composition.Functions[key].Name)
		}

		v := reflect.ValueOf(latestData.returnedInfos)
		t := reflect.TypeOf(latestData.returnedInfos)

		// Uso la reflection per iterare sui campi di QueryInformations
		activeTermsValue := reflect.ValueOf(activeTerms) // Reflection su activeTerms

		for i := 0; i < v.NumField(); i++ {
			fieldName := t.Field(i).Name // Nome del campo (es. AvgFcRespTime)
			fieldValue := v.Field(i)     // Valore del campo

			// Usa reflection per controllare se il campo è attivo in activeTerms
			activeField := activeTermsValue.FieldByName(fieldName)
			if activeField.IsValid() && activeField.Bool() { // Controlla se il campo esiste e se è true
				fmt.Printf("Campo '%s' è attivo in activeTerms\n", fieldName)

				if fieldValue.Kind() == reflect.Map {
					fmt.Printf("Campo: %s\n", fieldName)

					// Controlla se la mappa non è nil
					if !fieldValue.IsNil() {
						// Controlla se esiste la chiave nella mappa
						mapValue := fieldValue.MapIndex(reflect.ValueOf(key))
						if mapValue.IsValid() {
							//ho il valore per la metrica attiva in activeTerms per la funzione in esame
							fmt.Printf("  Chiave '%s' trovata, Valore: %v\n", key, mapValue)
						} else {
							fmt.Printf("  Chiave '%s' non trovata\n", key)
						}
					} else {
						fmt.Printf("  La mappa è nil\n")
					}
				} else {
					fmt.Printf("Campo %s non è una mappa\n", fieldName)
				}
			} else {
				fmt.Printf("Campo '%s' non è attivo in activeTerms\n", fieldName)
			}
		}
	}*/

	for key := range fr.composition.Functions {
		fmt.Println("------------------------------------------")
		fmt.Println("Function: ", fr.composition.Functions[key].Name)
		fmt.Println("------------------------------------------")

		// Controlla se la funzione è già usata in un altro workflow
		if contains(otherWorkFunc, fr.composition.Functions[key].Name) {
			fmt.Printf("Funzione %s già usata in un altro workflow\n", fr.composition.Functions[key].Name)
		}

		v := reflect.ValueOf(latestData.returnedInfos)
		t := reflect.TypeOf(latestData.returnedInfos)

		// Uso la reflection per iterare sui campi di QueryInformations
		policyDefValue := reflect.ValueOf(policyDef) // Reflection su PolicyDefinition

		for i := 0; i < v.NumField(); i++ {
			fieldName := t.Field(i).Name // Nome del campo (es. AvgFcRespTime)
			fieldValue := v.Field(i)     // Valore del campo

			// Usa reflection per ottenere il campo corrispondente in PolicyDefinition
			policyField := policyDefValue.FieldByName(fieldName)
			if policyField.IsValid() && policyField.Kind() == reflect.Slice {
				// Itera su tutti gli elementi della slice ([]PolicyElem)
				for j := 0; j < policyField.Len(); j++ {
					policyElem := policyField.Index(j).Interface().(policyElem) // Elemento corrente

					if policyElem.isAct { // Controlla se il campo è attivo
						fmt.Printf("Campo '%s' è attivo con soglia: %f\n", fieldName, policyElem.threshold)

						if fieldValue.Kind() == reflect.Map {
							fmt.Printf("Campo: %s\n", fieldName)

							// Controlla se la mappa non è nil
							if !fieldValue.IsNil() {
								// Controlla se esiste la chiave nella mappa
								mapValue := fieldValue.MapIndex(reflect.ValueOf(key))
								if mapValue.IsValid() {
									// Ottieni il valore per la metrica
									fmt.Printf("  Chiave '%s' trovata, Valore: %v\n", key, mapValue)

									// Confronta il valore della metrica con la soglia
									metricValue := mapValue.Float() // Assumendo che sia float64
									if metricValue > policyElem.threshold {
										fmt.Printf("  Valore %v supera la soglia %f\n", metricValue, policyElem.threshold)
									} else {
										fmt.Printf("  Valore %v non supera la soglia %f\n", metricValue, policyElem.threshold)
									}
								} else {
									fmt.Printf("  Chiave '%s' non trovata\n", key)
								}
							} else {
								fmt.Printf("  La mappa è nil\n")
							}
						} else {
							fmt.Printf("Campo %s non è una mappa\n", fieldName)
						}
					} else {
						fmt.Printf("Campo '%s' non è attivo in questo elemento di PolicyDefinition\n", fieldName)
					}
				}
			} else {
				fmt.Printf("Campo '%s' non trovato in PolicyDefinition o non è una slice\n", fieldName)
			}
		}
	}

	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		ok, _ := FuseFc(fr.composition)
		if !ok {
			fr.returnChannel <- fusionResult{action: NOOP}
		} else {
			fmt.Println("new functions vector is ", fr.composition.Functions)
			fr.returnChannel <- fusionResult{action: FUSED}
		}
	} else {
		fmt.Println("DON'T FUSE HERE with total dataMap: ", dataMap)
		fr.returnChannel <- fusionResult{action: NOOP}
	}

}

func saveInfos(infos returnedOutputData) {
	//saveInfos
	dataMap[infos.timestamp] = infos
}

func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func retrieveWorkflowsFunctions(currentWorkFlow string) ([]string, error) {
	//saveInfos
	workflows, error := function.GetAllWithPrefix("/fc")
	if error != nil {
		return nil, fmt.Errorf("Retrieving workflows error")
	}

	uniqueFunctions := make(map[string]struct{})

	for _, s := range workflows {
		fmt.Printf("Workflow: %s\n", s)
		if s == currentWorkFlow {
			log.Println("Non analizzare funzioni del workflow corrente")
			continue
		}
		funComp, ok := fc.GetFC(s)
		if !ok {
			return nil, fmt.Errorf("Dropping request for unknown FC '%s'", s)
		}

		// Itera sulle funzioni della composizione
		for funcName := range funComp.Functions {
			// Aggiungi il nome della funzione alla mappa
			uniqueFunctions[funcName] = struct{}{}
		}
	}

	var result []string
	for funcName := range uniqueFunctions {
		result = append(result, funcName)
	}

	return result, nil
}
