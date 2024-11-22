package fc_fusion

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
)

var metricInfos chan ReturnedOutputData
var dataMap map[time.Time]ReturnedOutputData

func Run(p FusionPolicy) {
	metricInfos = make(chan ReturnedOutputData, 500)
	dataMap = make(map[time.Time]ReturnedOutputData)

	// initialize Resources
	//forse info saranno utili nel corso della vita del componente di fusione
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current resources for fusion: %v\n", &node.Resources)

	// initialize fusion policy
	p.Init()
	log.Println("Fusion Component started.")

	var r ReturnedOutputData
	for {
		select {
		case r = <-metricInfos: // receive composition infos
			go p.OnArrival(r)
		}
	}

}

func SubmitInfos(data ReturnedOutputData) error {
	metricInfos <- data // send infos

	return nil
}

func fusionDecide(infos ReturnedOutputData) {
	//saveInfos
	dataMap[infos.Timestamp] = infos
	//Dummy
	condition := true //MUST be determined by an appropriate policy analyzing the report
	if condition {
		for key := range dataMap {
			fmt.Println("------------------------------------------")
			fmt.Println("Timestamp Key: ", key)
			fmt.Println("Metrics: ", dataMap[key])
			fmt.Println("------------------------------------------")
		}
		fmt.Println("")
	} else {
		fmt.Println("DON'T FUSE HERE with total dataMap: ", dataMap)
	}
}
