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
