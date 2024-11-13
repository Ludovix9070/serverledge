package fc_fusion

import (
	"github.com/grussorusso/serverledge/internal/fc"
)

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
