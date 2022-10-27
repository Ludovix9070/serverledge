package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"math/rand"
	"time"
)

const (
	LOCAL     = 0
	OFFLOADED = 1
)

const (
	DROP_REQUEST    = 0
	EXECUTE_REQUEST = 1
	OFFLOAD_REQUEST = 2
)

var startingExecuteProb = 0.5
var startingOffloadProb = 0.5
var evaluationInterval = 5

var rGen *rand.Rand

// TODO consider classes
type functionInfo struct {
	name string
	//Number of function requests
	count [2]int
	//Mean duration time
	meanDuration [2]float64
	//Variance of the duration time
	varianceDuration [2]float64
	//Number of requests that missed the deadline
	missed int
	//Offload latency
	offloadTime float64
	//Average of init times when cold start
	initTime float64
	//TODO consider classes
	probExecute float64
	probOffload float64
	probDrop    float64
	//
	arrivals        float64
	invokingClasses []function.QoSClass
}

type completedRequest struct {
	*function.Request
	location int
}

type arrivalRequest struct {
	function string
	class    string
}

var m = make(map[string]map[string]*functionInfo)

// TODO edit buffer?
var arrivalChannel = make(chan arrivalRequest, 10)

var requestChannel = make(chan completedRequest, 10)

func Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	class := r.ClassService

	prob := rGen.Float64()

	log.Printf("Request with class %s#%f#%f#%f\n", class.Name,
		class.Utility, r.GetMaxRT(), class.CompletedPercentage)

	var pe float64
	var po float64
	var pd float64

	arrivalChannel <- arrivalRequest{function: name, class: class.Name}

	fInfo, prs := m[name][class.Name]
	if !prs {
		pe = startingExecuteProb
		po = startingOffloadProb
		pd = 1 - (pe + po)
	} else {
		pe = fInfo.probExecute
		po = fInfo.probOffload
		pd = fInfo.probDrop
	}

	//warmNumber, isWarm := node.WarmStatus()[name]
	if !r.CanDoOffloading {
		pd = pd / (pd + pe)
		pe = pe / (pd + pe)
		po = 0
	} else if node.Resources.AvailableCPUs < r.Fun.CPUDemand &&
		node.Resources.AvailableMemMB < r.Fun.MemoryMB {
		pd = pd / (pd + po)
		po = po / (pd + po)
		pe = 0
	}

	if prob <= pe {
		log.Println("Execute LOCAL")
		return EXECUTE_REQUEST
	} else if prob <= pe+po {
		log.Println("Execute OFFLOAD")
		return OFFLOAD_REQUEST
	} else {
		log.Println("Execute DROP")
		return DROP_REQUEST
	}
}

func InitDecisionEngine() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	go ShowData()
	go handler()
}

func handler() {
	evaluationTicker :=
		time.NewTicker(time.Duration(evaluationInterval) * time.Second)

	for {
		select {
		case _ = <-evaluationTicker.C:
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Evaluating")
			for f, functionMap := range m {
				for c, finfo := range functionMap {
					log.Printf("Arrival of %s-%s: %f\n", f, c, finfo.arrivals/float64(evaluationInterval))
				}
			}

			updateProbabilities()

			//Reset Map
			for _, functionMap := range m {
				for _, finfo := range functionMap {
					finfo.arrivals = 0
				}
			}
		case r := <-requestChannel:
			updateData(r)
		case arr := <-arrivalChannel:
			fMap, prs := m[arr.function]
			if !prs {
				m[arr.function] = make(map[string]*functionInfo)
			}

			fInfo, prs := fMap[arr.class]
			if !prs {
				fInfo = &functionInfo{name: arr.function,
					probExecute: startingExecuteProb,
					probOffload: startingOffloadProb,
					probDrop:    1 - (startingExecuteProb + startingOffloadProb),
					arrivals:    0}
			}

			fInfo.arrivals++
			m[arr.function][arr.class] = fInfo
		}
	}
}

func updateProbabilities() {

}

func ShowData() {
	for {
		time.Sleep(5 * time.Second)
		for _, functionMap := range m {
			for _, finfo := range functionMap {
				log.Println(finfo)
			}
		}
	}
}

func Completed(r *function.Request, offloaded int) {
	requestChannel <- completedRequest{
		Request:  r,
		location: offloaded,
	}
}

// Delete TODO handle delete, delete from other nodes?
func Delete(name string) {
	delete(m, name)
}

// UpdateDataAsync TODO handle async interaction
func UpdateDataAsync(resp function.Response, reqId string) {

}

func updateData(r completedRequest) {
	name := r.Fun.Name
	class := r.ClassService.Name

	location := r.location

	fInfo, prs := m[name][class]
	if !prs {
		log.Println("Not exist")
		fInfo = &functionInfo{name: name,
			probExecute: startingExecuteProb,
			probOffload: startingOffloadProb,
			probDrop:    1 - (startingExecuteProb + startingOffloadProb)}
	}

	fInfo.count[location] = fInfo.count[location] + 1

	//Welford mean and variance
	diff := r.ExecReport.Duration - fInfo.meanDuration[location]
	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
		(1/float64(fInfo.count[location]))*(diff)
	diff2 := r.ExecReport.Duration - fInfo.meanDuration[location]

	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])

	if !r.ExecReport.IsWarmStart {
		diff := r.ExecReport.InitTime - fInfo.initTime
		fInfo.initTime = fInfo.initTime +
			(1/float64(fInfo.count[location]))*(diff)
	}

	if r.ExecReport.OffloadLatency != 0 {
		diff := r.ExecReport.OffloadLatency - fInfo.offloadTime
		fInfo.offloadTime = fInfo.offloadTime +
			(1/float64(fInfo.count[location]))*(diff)
	}

	if r.ExecReport.ResponseTime > r.GetMaxRT() {
		fInfo.missed++
	}
}
