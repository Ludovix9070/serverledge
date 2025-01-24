package fc_fusion

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"runtime"
	"sort"
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

func fusionEvaluate(fr *fusionRequest, policyDef policyDefinitionTerms) {
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

	var functionVector []functionElem

	//genero vettore funzioni del workflow e ne valuto la fusibilità se valore nella policy abilitato
	if policyDef.BlockSharedFunc.isAct {
		//se funzione già usata in altri workflows, non fondo
		otherWorkFunc, error := retrieveWorkflowsFunctions(fr.composition.Name)
		if error != nil {
			log.Println(error)
			fr.returnChannel <- fusionResult{action: NOOP}
		}

		log.Println(otherWorkFunc)

		for key := range fr.composition.Functions {
			elem := functionElem{name: fr.composition.Functions[key].Name}
			if contains(otherWorkFunc, fr.composition.Functions[key].Name) {
				elem.canBeFused = false
				fmt.Printf("Funzione %s già usata in un altro workflow\n", fr.composition.Functions[key].Name)
			} else {
				elem.canBeFused = true
			}
			functionVector = append(functionVector, elem)
		}
	} else {
		for key := range fr.composition.Functions {
			elem := functionElem{name: fr.composition.Functions[key].Name, canBeFused: true}
			functionVector = append(functionVector, elem)
		}
	}

	fmt.Println(functionVector)

	//se le funzioni da valutare hanno già una durata molto ampia, non fondo
	if policyDef.MaxFuncDuration.isAct {
		fmt.Println("MaxFuncDuration è attivo. Confronto con threshold:", policyDef.MaxFuncDuration.threshold)

		v := reflect.ValueOf(latestData.returnedInfos) // Valori delle metriche
		//t := reflect.TypeOf(latestData.returnedInfos)  // Tipo delle metriche

		for i := range functionVector {
			if !functionVector[i].canBeFused {
				// Salto la funzione se non può essere fusa
				fmt.Printf("Funzione '%s' già marcata come non fondibile. Salto controllo.\n", functionVector[i].name)
				continue
			}

			funcName := functionVector[i].name
			fmt.Printf("Valutazione della funzione '%s'...\n", funcName)

			fieldValue := v.FieldByName("AvgFunDurationTime") // Cerco AvgFunDurationTime
			if !fieldValue.IsValid() || fieldValue.Kind() != reflect.Map {
				fmt.Println("Campo AvgFunDurationTime non trovato o non è una mappa.")
				continue
			}

			// Controllo se la mappa non è nil
			if fieldValue.IsNil() {
				fmt.Printf("Campo AvgFunDurationTime per funzione '%s' è nil.\n", funcName)
				continue
			}

			// Cerco il valore nella mappa
			mapValue := fieldValue.MapIndex(reflect.ValueOf(funcName))
			if mapValue.IsValid() {
				metricValue := mapValue.Float()
				fmt.Printf("  Valore di AvgFunDurationTime per '%s': %f\n", funcName, metricValue)

				// Confronto con la threshold
				thresh := calculateDurationThreshold(latestData.returnedInfos.AvgFunDurationTime, policyDef.MaxFuncDuration.threshold)
				//if metricValue > policyDef.MaxFuncDuration.threshold[0] {
				if metricValue > thresh {
					fmt.Printf("  Valore %f supera la soglia %f. Imposto canBeFused a false.\n", metricValue, thresh)
					functionVector[i].canBeFused = false // Imposto canBeFused a false
				} else {
					fmt.Printf("  Valore %f NON supera la soglia %f. Imposto canBeFused a true.\n", metricValue, thresh)
					functionVector[i].canBeFused = true // Imposto canBeFused a false
				}

			} else {
				fmt.Printf("  Nessun valore trovato per AvgFunDurationTime della funzione '%s'.\n", funcName)
			}
		}
	}

	fmt.Println(functionVector)

	if policyDef.DurInit.isAct {
		fmt.Println("DurInit è attivo. Confronto con threshold:", policyDef.DurInit.threshold)

		v := reflect.ValueOf(latestData.returnedInfos) // Valori delle metriche
		//t := reflect.TypeOf(latestData.returnedInfos)  // Tipo delle metriche

		for i := range functionVector {
			if !functionVector[i].canBeFused {
				// Salto la funzione se non può essere fusa
				fmt.Printf("Funzione '%s' già marcata come non fondibile. Salto controllo.\n", functionVector[i].name)
				continue
			}

			funcName := functionVector[i].name
			fmt.Printf("Valutazione della funzione '%s' per DurInit...\n", funcName)

			// Cerco AvgFunDurationTime
			durationField := v.FieldByName("AvgFunDurationTime")
			if !durationField.IsValid() || durationField.Kind() != reflect.Map {
				fmt.Println("Campo AvgFunDurationTime non trovato o non è una mappa.")
				continue
			}

			// Cerco AvgFunInitTime
			initField := v.FieldByName("AvgFunInitTime")
			if !initField.IsValid() || initField.Kind() != reflect.Map {
				fmt.Println("Campo AvgFunInitTime non trovato o non è una mappa.")
				continue
			}

			// Controllo se le mappe non sono nil
			if durationField.IsNil() || initField.IsNil() {
				fmt.Printf("Campo AvgFunDurationTime o AvgFunInitTime è nil per funzione '%s'.\n", funcName)
				continue
			}

			// Cerco i valori delle metriche per la funzione
			durationValue := durationField.MapIndex(reflect.ValueOf(funcName))
			initValue := initField.MapIndex(reflect.ValueOf(funcName))

			if durationValue.IsValid() && initValue.IsValid() {
				// Calcolo il rapporto
				durationMetric := durationValue.Float()
				initMetric := initValue.Float()

				if initMetric == 0 {
					fmt.Printf("  Inizializzazione (AvgFunInitTime) per '%s' è 0, impossibile calcolare il rapporto.\n", funcName)
					continue
				}

				ratio := durationMetric / initMetric
				fmt.Printf("  Rapporto AvgFunDurationTime/AvgFunInitTime per '%s': %f\n", funcName, ratio)

				// Confronta il rapporto con la soglia
				if ratio > policyDef.DurInit.threshold[0] {
					fmt.Printf("  Rapporto %f supera la soglia %f. Imposto canBeFused a false.\n", ratio, policyDef.DurInit.threshold)
					functionVector[i].canBeFused = false // Imposta canBeFused a false
				} else {
					fmt.Printf("  Rapporto %f NON supera la soglia %f. Imposto canBeFused a true.\n", ratio, policyDef.DurInit.threshold)
					functionVector[i].canBeFused = true // Imposta canBeFused a false
				}
			} else {
				fmt.Printf("  Valori mancanti per AvgFunDurationTime o AvgFunInitTime per la funzione '%s'.\n", funcName)
			}
		}
	}

	fmt.Println(functionVector)

	condition := false
	//se tutte le funzioni nel workflow non fondibili, evito controlli post sulla fusibilità
	for _, v := range functionVector {
		if v.canBeFused {
			condition = true
			break
		}
	}

	if condition {
		ok, _ := FuseFcEvaluate(fr.composition, policyDef, functionVector)
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

func calculateDurationThreshold(vectorInfo map[string]float64, weights []float64) float64 {
	values := make([]float64, 0, len(vectorInfo))
	for _, v := range vectorInfo {
		values = append(values, v)
	}

	fmt.Println(values)

	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)
	percentile90 := calculatePercentile(values, 90.0)

	k := 2.0 // Fattore per pesare la deviazione standard
	stdDevLength := stdDevToLength(mean, stdDev, k)

	// Definisci i pesi
	//weights := [3]float64{0.4, 0.3, 0.3}

	// Calcola la soglia combinata
	threshold := calculateWeightedThreshold(mean, stdDevLength, percentile90, weights)

	// Stampa i risultati
	fmt.Printf("Media: %f\n", mean)
	fmt.Printf("Deviazione standard sommata a media con k %f: %f\n", k, stdDevLength)
	fmt.Printf("90° percentile: %f\n", percentile90)
	fmt.Printf("Threshold combinata: %f\n", threshold)

	return threshold
}

// Funzione per calcolare media
func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// Funzione per calcolare deviazione standard
func calculateStdDev(values []float64, mean float64) float64 {
	var varianceSum float64
	for _, v := range values {
		varianceSum += math.Pow(v-mean, 2)
	}
	variance := varianceSum / float64(len(values))
	return math.Sqrt(variance)
}

// Funzione per trasformare deviazione standard in una misura di lunghezza
func stdDevToLength(mean, stdDev, factor float64) float64 {
	return mean + factor*stdDev // Trasforma la deviazione standard in una lunghezza
}

// Funzione per calcolare il 90° percentile
func calculatePercentile(values []float64, percentile float64) float64 {
	sort.Float64s(values) // Ordina i valori
	index := int(math.Ceil(percentile/100*float64(len(values)))) - 1
	return values[index]
}

// Funzione per combinare media, deviazione standard e 90° percentile con pesi
func calculateWeightedThreshold(mean, stdDev, percentile90 float64, weights []float64) float64 {
	return weights[0]*mean + weights[1]*stdDev + weights[2]*percentile90
}
