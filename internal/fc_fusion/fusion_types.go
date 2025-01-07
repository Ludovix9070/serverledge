package fc_fusion

import (
	"time"

	"github.com/grussorusso/serverledge/internal/fc"
)

type ReturnedOutputData struct {
	ReturnedInfos QueryInformations
	Timestamp     time.Time
}

type QueryInformations struct {
	AvgTotalColdStartsTime map[string]float64
	AvgFcRespTime          map[string]float64
	AvgFunDurationTime     map[string]float64
	AvgOutputFunSize       map[string]float64
	AvgFunInitTime         map[string]float64
}

type PolicyTerms struct {
	AvgTotalColdStartsTime bool
	AvgFcRespTime          bool
	AvgFunDurationTime     bool
	AvgOutputFunSize       bool
	AvgFunInitTime         bool
}

/*type PolicyEvaluatedData struct {
	ReturnedInfos QueryInformations
	Act     bool
}

type PolicyTerms struct {
	ActivatedTerms map[string]PolicyEvaluatedData
}*/

// struttura contenente le informazioni necessarie per la fusione prese dopo una esecuzione
type fusionRequest struct {
	Composition   *fc.FunctionComposition
	returnChannel chan fusionResult
}

type fusionResult struct {
	action action
}

// sono le costanti per le azioni di fusione
const (
	FUSED action = 0
	NOOP         = 1
)

type action int64
