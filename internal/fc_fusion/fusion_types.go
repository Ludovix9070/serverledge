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

// struct che contiene i termini della policy
type policyDefinitionTerms struct {
	MaxFuncDuration policyElem //pre
	MaxDimPkt       policyElem //post
	DurInit         policyElem //pre
	MaxMemoryDelta  policyElem //post
	MaxCpuDelta     policyElem //post
	BlockSharedFunc policyElem //pre
}

// ogni elemento della policy può essere attivo/non attivo e con la rispettiva threshold
type policyElem struct {
	isAct     bool
	threshold []float64
}

type functionElem struct {
	name       string
	canBeFused bool
}

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
