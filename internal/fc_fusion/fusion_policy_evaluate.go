package fc_fusion

import (
	"log"

	//"runtime/metrics"

	"github.com/grussorusso/serverledge/internal/config"
)

/*var terms = PolicyTerms{
	AvgTotalColdStartsTime: false,
	AvgFcRespTime:          false,
	AvgFunDurationTime:     true,
	AvgOutputFunSize:       true,
	AvgFunInitTime:         true,
}*/

/*var policyDef = policyDefinition{
	AvgTotalColdStartsTime: []policyElem{{isAct: true, threshold: 2.0}},
	AvgFcRespTime:          []policyElem{{isAct: false, threshold: 1.5}},
	AvgFunDurationTime:     []policyElem{{isAct: true, threshold: 5.0}},
	AvgOutputFunSize:       []policyElem{{isAct: true, threshold: 100.0}},
	AvgFunInitTime:         []policyElem{{isAct: true, threshold: 2.0}},
}*/

var policyDefs = policyDefinitionTerms{
	MaxFuncDuration: policyElem{isAct: true, threshold: []float64{0.4, 0.3, 0.3}}, //pre, threshold in percentage
	MaxDimPkt:       policyElem{isAct: true, threshold: []float64{1.0}},           //post, threshold in MB, max in etcd
	DurInit:         policyElem{isAct: true, threshold: []float64{0.5}},           //pre, abs
	MaxMemoryDelta:  policyElem{isAct: true, threshold: []float64{0.3}},           //post, threshold in percentage
	MaxCpuDelta:     policyElem{isAct: true, threshold: []float64{0.3}},           //post, threshold in percentage
	BlockSharedFunc: policyElem{isAct: true, threshold: []float64{0.0}},           //pre,
}

type EvaluateFusionPolicy struct {
	queue queue
}

func (p *EvaluateFusionPolicy) Init() {
	queueCapacity := config.GetInt(config.FUSION_QUEUE_CAPACITY, 0)
	if queueCapacity > 0 {
		log.Printf("Configured fusion queue with capacity %d\n", queueCapacity)
		p.queue = NewFIFOQueue(queueCapacity)
	} else {
		p.queue = nil
	}
}

func (p *EvaluateFusionPolicy) OnCompletion(fr *fusionRequest) {

}

// OnArrival for default fusion policy is called every time a dag execution terminates
func (p *EvaluateFusionPolicy) OnArrival(fr *fusionRequest) {
	//Solo per Debug
	//fmt.Printf("ON ARRIVAL DEFAULT WITH REPORT RESULT %v\n", info.ExecReport.Result)

	/*//fusionDecide(*info)
	if info != nil {
		saveInfos(*info)
		fusionDecide()
	}*/
	fusionEvaluate(fr, policyDefs)

}
