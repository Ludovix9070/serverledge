package fc_fusion

import (
	"time"

	"github.com/grussorusso/serverledge/internal/fc"
)

type returnedOutputData struct {
	returnedInfos queryInformations
	timestamp     time.Time
}

type queryInformations struct {
	AvgTotalColdStartsTime map[string]float64
	AvgFcRespTime          map[string]float64
	AvgFunDurationTime     map[string]float64
	AvgOutputFunSize       map[string]float64
	AvgFunInitTime         map[string]float64
}

/*
type PolicyTerms struct {
	avgTotalColdStartsTime bool
	avgFcRespTime          bool
	avgFunDurationTime     bool
	avgOutputFunSize       bool
	avgFunInitTime         bool
}*/

type policyDefinition struct {
	AvgTotalColdStartsTime []policyElem
	AvgFcRespTime          []policyElem
	AvgFunDurationTime     []policyElem
	AvgOutputFunSize       []policyElem
	AvgFunInitTime         []policyElem
}

type policyElem struct {
	isAct     bool
	threshold float64
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
	composition   *fc.FunctionComposition
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
