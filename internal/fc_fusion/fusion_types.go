package fc_fusion

import (
	"time"

	"github.com/grussorusso/serverledge/internal/fc"
)

type ReturnedOutputData struct {
	AvgTotalColdStartsTime map[string]float64
	AvgFcRespTime          map[string]float64
	AvgFunDurationTime     map[string]float64
	AvgOutputFunSize       map[string]float64
	Timestamp              time.Time
}

// struttura contenente le informazioni necessarie per la fusione prese dopo una esecuzione
type fusionInfo struct {
	ExecReport      *fc.CompositionExecutionReport
	Composition     *fc.FunctionComposition
	decisionChannel chan fusionDecision
}

type fusionDecision struct {
	action action
}

// sono le costanti per le azioni di fusione
const (
	EVALUATE_FUSION action = 0
	NOOP                   = 1
)

type action int64
