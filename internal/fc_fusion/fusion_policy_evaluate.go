package fc_fusion

import (
	"log"

	//"runtime/metrics"

	"github.com/grussorusso/serverledge/internal/config"
)

var terms = PolicyTerms{
	AvgTotalColdStartsTime: false,
	AvgFcRespTime:          false,
	AvgFunDurationTime:     true,
	AvgOutputFunSize:       true,
	AvgFunInitTime:         true,
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
	fusionEvaluate(fr, terms)

}
