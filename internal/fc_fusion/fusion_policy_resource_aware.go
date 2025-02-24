package fc_fusion

import (
	"log"

	//"runtime/metrics"

	"github.com/grussorusso/serverledge/internal/config"
)

var policyDefsResourceAware = policyDefinitionTerms{
	MaxFuncDuration: policyElem{isAct: true, threshold: []float64{30.0}}, //pre e post, threshold in timeout seconds
	MaxDimPkt:       policyElem{isAct: true, threshold: []float64{1.0}},  //post, threshold in MB, max in etcd
	DurInit:         policyElem{isAct: true, threshold: []float64{0.5}},  //pre, abs
	MaxMemoryDelta:  policyElem{isAct: true, threshold: []float64{0.3}},  //post, threshold in percentage
	MaxCpuDelta:     policyElem{isAct: true, threshold: []float64{0.3}},  //post, threshold in percentage
	BlockSharedFunc: policyElem{isAct: true, threshold: []float64{1.0}},  //pre,
}

type ResourceAwareFusionPolicy struct {
	queue queue
}

func (p *ResourceAwareFusionPolicy) Init() {
	queueCapacity := config.GetInt(config.FUSION_QUEUE_CAPACITY, 0)
	if queueCapacity > 0 {
		log.Printf("Configured fusion queue with capacity %d\n", queueCapacity)
		p.queue = NewFIFOQueue(queueCapacity)
	} else {
		p.queue = nil
	}
}

func (p *ResourceAwareFusionPolicy) OnCompletion(fr *fusionRequest) {

}

// OnArrival for default fusion policy is called every time a dag execution terminates
func (p *ResourceAwareFusionPolicy) OnArrival(fr *fusionRequest) {
	fusionEvaluate(fr, policyDefsResourceAware)

}
