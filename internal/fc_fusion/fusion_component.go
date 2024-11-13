package fc_fusion

import (
	"fmt"
	"log"
	"runtime"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/node"
)

var infos chan *fusionInfo
var executionInfo map[string][]fc.CompositionExecutionReport

func Run(p FusionPolicy) {
	infos = make(chan *fusionInfo, 500)
	executionInfo = make(map[string][]fc.CompositionExecutionReport)

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

	var f *fusionInfo
	for {
		select {
		case f = <-infos: // receive composition infos
			go p.OnArrival(f)
		}
	}

}

// SubmitFusionInfo gets composition execution informations to evaluate fusion
func SubmitFusionInfos(report *fc.CompositionExecutionReport, funComp *fc.FunctionComposition) error {
	fusionInfo := fusionInfo{
		ExecReport:      report,
		Composition:     funComp,
		decisionChannel: make(chan fusionDecision, 1)}
	infos <- &fusionInfo // send infos

	fusionDecision, ok := <-fusionInfo.decisionChannel
	if !ok {
		return fmt.Errorf("could not get the fusion decision")
	}

	if fusionDecision.action == EVALUATE_FUSION {
		//Solo per Debug
		fmt.Println("MAKE FUSION HERE, CALL funcs in fusion.go")
		//qui devo effettuare la fusione del dag
	}

	return nil
}

func fusionDecide(infos *fusionInfo) {
	saveInfos(infos)
	//Solo per Debug
	//fmt.Printf("DECIDE IF TO FUSE HERE WITH REPORT RESULT %v\n", infos.ExecReport.Result)
	//fmt.Printf("DECIDE IF TO FUSE FOR FC %s\n", infos.Composition.Name)

	//Dummy
	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		decision := fusionDecision{action: EVALUATE_FUSION}
		infos.decisionChannel <- decision
	} else {
		decision := fusionDecision{action: NOOP}
		infos.decisionChannel <- decision
	}

}

func saveInfos(infos *fusionInfo) {
	// Funzione per aggiungere un report alla mappa
	addReport := func(key string, report fc.CompositionExecutionReport) {
		// Controlla se la chiave è già presente nella mappa
		if _, exists := executionInfo[key]; exists {
			// Se esiste, fai l'append dell'elemento alla slice esistente
			executionInfo[key] = append(executionInfo[key], report)
		} else {
			// Se non esiste, crea una nuova slice e aggiungi l'elemento
			executionInfo[key] = []fc.CompositionExecutionReport{report}
		}
	}
	addReport(infos.Composition.Name, *infos.ExecReport)

	// Solo per Debug
	fmt.Println("Execution Info Map (only Result Field):")
	for key, reports := range executionInfo {
		fmt.Printf("Key: %s\n", key)
		for _, report := range reports {
			fmt.Printf("  Result: %v\n", report.Result)
		}
	}
}
