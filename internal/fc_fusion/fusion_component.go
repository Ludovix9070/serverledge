package fc_fusion

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/node"
)

var metricInfos chan *ReturnedOutputData

// var fusionInvocationChannel chan *fc.FunctionComposition
var fusionInvocationChannel chan *fusionRequest
var dataMap map[time.Time]ReturnedOutputData

func Run(p FusionPolicy) {
	metricInfos = make(chan *ReturnedOutputData, 500)
	fusionInvocationChannel = make(chan *fusionRequest, 500)
	dataMap = make(map[time.Time]ReturnedOutputData)

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

func SubmitInfos(data ReturnedOutputData) error {
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
		Composition:   fc,
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
	fmt.Println("FUSE COMMAND with Default/Alwaysfuse policy for composition ", fr.Composition.Name)
	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		for key := range dataMap {
			fmt.Println("------------------------------------------")
			fmt.Println("Timestamp Key: ", key)
			fmt.Println("Metrics: ", dataMap[key].ReturnedInfos)
			fmt.Println("------------------------------------------")
		}
		fmt.Println("")

		ok, _ := FuseFc(fr.Composition)
		if !ok {
			fr.returnChannel <- fusionResult{action: NOOP}
		} else {
			fmt.Println("new functions vector is ", fr.Composition.Functions)
			fr.returnChannel <- fusionResult{action: FUSED}
		}
	} else {
		fmt.Println("DON'T FUSE HERE with total dataMap: ", dataMap)
		fr.returnChannel <- fusionResult{action: NOOP}
	}

}

func fusionEvaluate(fr *fusionRequest, activeTerms PolicyTerms) {
	//saveInfos
	//for all the fc? TODO
	//dataMap[infos.Timestamp] = infos
	//Dummy
	fmt.Println("FUSE COMMAND with Evaluate Policy for composition ", fr.Composition.Name)
	for key := range dataMap {
		fmt.Println("------------------------------------------")
		fmt.Println("Timestamp Key: ", key)
		fmt.Println("Metrics: ", dataMap[key].ReturnedInfos)
		fmt.Println("------------------------------------------")
	}
	fmt.Println("")

	var latestTime time.Time
	var latestData ReturnedOutputData

	for timestamp, data := range dataMap {
		if timestamp.After(latestTime) {
			latestTime = timestamp
			latestData = data
			fmt.Println(latestData)
		}
	}

	/*for key := range fr.Composition.Functions {
		fmt.Println("------------------------------------------")
		fmt.Println("Function: ", fr.Composition.Functions[key])
		fmt.Println("------------------------------------------")
		v := reflect.ValueOf(latestData.ReturnedInfos)
		t := reflect.TypeOf(latestData.ReturnedInfos)

		//sto ciclando sui campi di QueryInformations
		for i := 0; i < v.NumField(); i++ {
			fieldName := t.Field(i).Name // Nome del campo es.AvgFcRespTime
			fieldValue := v.Field(i)     // Valore del campo

			if fieldValue.Kind() == reflect.Map {
				fmt.Printf("Campo: %s\n", fieldName)

				// Controlla se la mappa non è nil
				if !fieldValue.IsNil() {
					// Controlla se esiste la chiave nella mappa
					mapValue := fieldValue.MapIndex(reflect.ValueOf(key))
					if mapValue.IsValid() {
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
		}

	}*/

	for key := range fr.Composition.Functions {
		fmt.Println("------------------------------------------")
		fmt.Println("Function: ", fr.Composition.Functions[key])
		fmt.Println("------------------------------------------")

		v := reflect.ValueOf(latestData.ReturnedInfos)
		t := reflect.TypeOf(latestData.ReturnedInfos)

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
	}

	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		ok, _ := FuseFc(fr.Composition)
		if !ok {
			fr.returnChannel <- fusionResult{action: NOOP}
		} else {
			fmt.Println("new functions vector is ", fr.Composition.Functions)
			fr.returnChannel <- fusionResult{action: FUSED}
		}
	} else {
		fmt.Println("DON'T FUSE HERE with total dataMap: ", dataMap)
		fr.returnChannel <- fusionResult{action: NOOP}
	}

}

func saveInfos(infos ReturnedOutputData) {
	//saveInfos
	dataMap[infos.Timestamp] = infos
}
