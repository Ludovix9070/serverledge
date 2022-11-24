package scheduling

import (
	"context"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	pb "github.com/grussorusso/serverledge/internal/scheduling/protobuf"
	"google.golang.org/grpc"
	"log"
)

func solve(m map[string]*functionInfo) {
	if len(m) == 0 {
		return
	}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())

	serverAddr := config.GetString(config.SOLVER_ADDRESS, "localhost:2500")

	conn, err := grpc.Dial(serverAddr, opts...)
	defer conn.Close()
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewSolverClient(conn)

	functionList := make([]*pb.Function, 0)

	//TODO do only once
	classList := make([]*pb.QosClass, 0)

	for fName, fInfo := range m {

		invocationList := make([]*pb.FunctionInvocation, 0)

		for _, cFInfo := range fInfo.invokingClasses {
			arrivals := float32(cFInfo.arrivals)

			invocationList = append(invocationList, &pb.FunctionInvocation{
				QosClass: &cFInfo.className,
				Arrivals: &arrivals,
			})
		}

		memory := int32(fInfo.memory)
		cpu := float32(fInfo.cpu)
		durationLocal := float32(fInfo.meanDuration[LOCAL])
		durationOffloaded := float32(fInfo.meanDuration[OFFLOADED])
		initTimeLocal := float32(fInfo.initTime[LOCAL])
		initTimeOffloaded := float32(fInfo.initTime[OFFLOADED])
		pcold := float32(fInfo.probCold[LOCAL])
		pcoldOffloaded := float32(fInfo.probCold[OFFLOADED])

		x := &pb.Function{
			Name:              &fName,
			Memory:            &memory,
			Cpu:               &cpu,
			Invocations:       invocationList,
			Duration:          &durationLocal,
			DurationOffloaded: &durationOffloaded,
			InitTime:          &initTimeLocal,
			InitTimeOffloaded: &initTimeOffloaded,
			Pcold:             &pcold,
			PcoldOffloaded:    &pcoldOffloaded,
		}

		functionList = append(functionList, x)
	}

	if len(classList) == 0 {
		for cName, class := range Classes {
			utility := float32(class.Utility)
			mrt := float32(class.MaximumResponseTime)
			completedPercentage := float32(class.CompletedPercentage)
			name := cName

			classList = append(classList, &pb.QosClass{
				Name:                &name,
				Utility:             &utility,
				MaxResponseTime:     &mrt,
				CompletedPercentage: &completedPercentage,
			})
		}
	}

	latency := float32(OffloadLatency)
	cost := float32(config.GetFloat(config.CLOUD_COST, 0.01))
	cpu := float32(node.Resources.MaxCPUs)
	mem := float32(node.Resources.MaxMemMB)
	response, err := client.Solve(context.Background(), &pb.Request{
		OffloadLatency: &latency,
		Functions:      functionList,
		Classes:        classList,
		Cost:           &cost,
		Cpu:            &cpu,
		Memory:         &mem,
	})

	if err != nil {
		log.Println(err)
	}

	log.Println("Evaluation took: ", response.GetTimeTaken())
	res := response.GetFResponse()

	for _, r := range res {
		fInfo, prs := m[r.GetName()]

		if !prs {
			log.Println("Error in assigning probabilities")
			continue
		}

		invokingClasses := fInfo.invokingClasses
		for _, x := range r.GetClassResponses() {
			cFInfo, prs := invokingClasses[x.GetName()]
			if !prs {
				log.Println("Error in assigning probabilities")
				continue
			}

			cFInfo.probExecute = float64(x.GetPe())
			cFInfo.probOffload = float64(x.GetPo())
			cFInfo.probDrop = float64(x.GetPd())
			cFInfo.share = float64(x.GetShare())
		}
	}

}

//
//import (
//	"github.com/draffensperger/golp"
//	"github.com/grussorusso/serverledge/internal/node"
//	"log"
//	"math"
//	"time"
//)
//
//var numberOfFunctionClass int
//var functionPColdMap map[string]int
//
//var debug = true
//
//type cFInfoWithClass struct {
//	*classFunctionInfo
//	class string
//}
//
//func getPColdIndex(name string) int {
//	return numberOfFunctionClass*4 + functionPColdMap[name]
//}
//
//func getPExecutionIndex(index int) int {
//	return index * 4
//}
//
//func getPOffloadIndex(index int) int {
//	return index*4 + 1
//}
//
//func getPDropIndex(index int) int {
//	return index*4 + 2
//}
//
//func getShareIndex(index int) int {
//	return index*4 + 3
//}
//
//func SolveprobabilitiesLegacy(m map[string]*functionInfo) {
//	if len(m) == 0 {
//		return
//	}
//
//	list := make([]cFInfoWithClass, 0)
//
//	objectiveFunctionEntries := make([]float64, 0)
//	memoryConstraintEntries := make([]golp.Entry, 0)
//	cpuConstraintEntries := make([]golp.Entry, 0)
//
//	classMap := make(map[string][]*classFunctionInfo)
//
//	functionPColdMap = make(map[string]int)
//	functionPdIndex := make(map[string]int)
//
//	functionNumber := len(m)
//
//	for _, fInfo := range m {
//		for class, cFInfo := range fInfo.invokingClasses {
//			list = append(list, cFInfoWithClass{cFInfo, class})
//
//			classFunctionList, prs := classMap[class]
//			if !prs {
//				classFunctionList = make([]*classFunctionInfo, 1)
//				classFunctionList[0] = cFInfo
//			} else {
//				classFunctionList = append(classFunctionList, cFInfo)
//			}
//
//			classMap[class] = classFunctionList
//		}
//	}
//
//	numberOfFunctionClass = len(list)
//
//	index := 0
//	for fName := range m {
//		functionPColdMap[fName] = numberOfFunctionClass*4 + index
//
//		index++
//	}
//
//	//4 for every function, class pair and one pcold for each function
//	lp := golp.NewLP(0, numberOfFunctionClass*4+functionNumber)
//
//	for i := range list {
//		//Probability constraints
//		cFInfo := list[i]
//
//		if debug {
//			lp.SetColName(getPExecutionIndex(i), "PE"+list[i].name+list[i].class)
//			lp.SetColName(getPOffloadIndex(i), "PO"+list[i].name+list[i].class)
//			lp.SetColName(getPDropIndex(i), "PD"+list[i].name+list[i].class)
//			lp.SetColName(getShareIndex(i), "X"+list[i].name+list[i].class)
//		}
//
//		//Probability constraints
//		//TODO needed if the sum is < 1?
//		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), 1.0}}, golp.LE, 1)
//		lp.AddConstraintSparse([]golp.Entry{{getPOffloadIndex(i), 1.0}}, golp.LE, 1)
//		lp.AddConstraintSparse([]golp.Entry{{getPDropIndex(i), 1.0}}, golp.LE, 1)
//
//		//Sum of pe + pd + po = 1
//		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), 1.0},
//			{getPOffloadIndex(i), 1.0}, {getPDropIndex(i), 1.0}}, golp.EQ, 1)
//
//		//Constraint for the scaling value
//		//pe*time*arrival <= scale
//		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), cFInfo.meanDuration[LOCAL] * cFInfo.arrivals},
//			{getShareIndex(i), -1}}, golp.LE, 0)
//
//		//Response time solution
//		if Classes[cFInfo.class].MaximumResponseTime != -1 {
//			lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), cFInfo.meanDuration[LOCAL]},
//				{getPOffloadIndex(i), OffloadLatency + cFInfo.meanDuration[OFFLOADED]},
//				{getPColdIndex(list[i].name), cFInfo.initTime[LOCAL]}},
//				golp.LE, Classes[cFInfo.class].MaximumResponseTime)
//		}
//
//		objectiveFunctionEntries = append(objectiveFunctionEntries,
//			[]float64{cFInfo.arrivals * Classes[cFInfo.class].Utility,
//				cFInfo.arrivals * Classes[cFInfo.class].Utility,
//				0,
//				0}...)
//
//		memoryConstraintEntries = append(memoryConstraintEntries, []golp.Entry{{getShareIndex(i), float64(cFInfo.memory)}}...)
//
//		//TODO functions can have 0 CPU demand?
//		if cFInfo.cpu != 0 {
//			cpuConstraintEntries = append(cpuConstraintEntries, []golp.Entry{{getShareIndex(i), cFInfo.cpu}}...)
//		}
//
//		functionPdIndex[cFInfo.name+cFInfo.class] = getPDropIndex(i)
//	}
//
//	//Class constraint
//	for k, classList := range classMap {
//		classConstraintEntries := make([]golp.Entry, 0)
//		arrivalSum := 0.0
//
//		for i := range classList {
//			classConstraintEntries =
//				append(classConstraintEntries, []golp.Entry{{functionPdIndex[classList[i].name+k], classList[i].arrivals}}...)
//
//			arrivalSum += classList[i].arrivals
//		}
//
//		lp.AddConstraintSparse(classConstraintEntries, golp.LE, (1-Classes[k].CompletedPercentage)*arrivalSum)
//	}
//
//	for fName, index := range functionPColdMap {
//		lp.AddConstraintSparse([]golp.Entry{{index, 1.0}}, golp.LE, 1)
//
//		if debug {
//			lp.SetColName(index, "PC"+fName)
//		}
//
//		objectiveFunctionEntries = append(objectiveFunctionEntries, 1)
//	}
//
//	if len(memoryConstraintEntries) > 0 {
//		lp.AddConstraintSparse(memoryConstraintEntries, golp.LE, float64(node.Resources.MaxMemMB))
//	}
//
//	if len(cpuConstraintEntries) > 0 {
//		lp.AddConstraintSparse(cpuConstraintEntries, golp.LE, node.Resources.MaxCPUs)
//	}
//
//	//Objective function
//	lp.SetObjFn(objectiveFunctionEntries)
//	lp.SetMaximize()
//
//	start := time.Now()
//	sol := lp.Solve()
//	elapsed := time.Since(start)
//
//	vars := lp.Variables()
//
//	for i := range list {
//		cFInfo := list[i]
//
//		cFInfo.probExecute = vars[getPExecutionIndex(i)]
//		cFInfo.probOffload = vars[getPOffloadIndex(i)]
//		cFInfo.probDrop = vars[getPDropIndex(i)]
//		cFInfo.share = vars[getShareIndex(i)]
//	}
//
//	for name, index := range functionPColdMap {
//		_, prs := m[name]
//		if !prs {
//			continue
//		}
//
//		m[name].probCold = vars[index]
//	}
//
//	if debug {
//		log.Println(lp.WriteToString())
//		log.Printf("Resolution took %s", elapsed)
//		log.Println("Var: ", vars)
//		log.Println("Sol type: ", sol)
//		log.Println("Optimum: ", lp.Objective())
//	}
//}
//
//func ErlangB(m int, a float64) float64 {
//	sum := 0.0
//	fact := 1.0
//
//	for i := 1.0; i <= float64(m); i++ {
//		fact *= i
//		sum += math.Pow(a, i) / fact
//	}
//
//	sum += 1
//
//	return math.Pow(sum, -1) * (math.Pow(a, float64(m)) / fact)
//}
//
//func SolveColdStart(m map[string]*functionInfo) map[string]int {
//	outMap := make(map[string]int)
//
//	numberOfFunctions := len(m)
//	if numberOfFunctions == 0 {
//		return outMap
//	}
//
//	for fName, fInfo := range m {
//		sum := 0.0
//		arrivals := 0.0
//		w := 0
//
//		for _, cFInfo := range fInfo.invokingClasses {
//			sum += cFInfo.share
//			arrivals += cFInfo.arrivals
//		}
//
//		log.Printf("ERLANG(%d, %f): %f\n", w, arrivals/fInfo.meanDuration[LOCAL], ErlangB(w, arrivals/fInfo.meanDuration[LOCAL]))
//		log.Println("PCF > ErlangB", fInfo.probCold, ErlangB(w, arrivals/fInfo.meanDuration[LOCAL]))
//		for fInfo.probCold > ErlangB(w, arrivals/fInfo.meanDuration[LOCAL]) && float64(w+1) < sum {
//			w += 1
//			log.Printf("ERLANG(%d, %f): %f\n", w, arrivals/fInfo.meanDuration[LOCAL], ErlangB(w, arrivals/fInfo.meanDuration[LOCAL]))
//		}
//
//		outMap[fName] = w
//	}
//
//	return outMap
//}
//
//func getDeadlineSatisfactionProb(location int, cFInfo cFInfoWithClass, deadline float64) float64 {
//	if deadline == -1 {
//		return 1
//	}
//
//	//TODO is the correct value for missing data?
//	if cFInfo.count[location] == 0 {
//		return 1
//	}
//
//	//If there isn't enough data return 1
//	if cFInfo.count[location] == 0 {
//		return 1
//	}
//
//	if location == LOCAL {
//		return exponentialCDF(cFInfo.meanDuration[location], deadline-cFInfo.probCold*cFInfo.initTime[location])
//	} else {
//		return exponentialCDF(cFInfo.meanDuration[location], deadline-cFInfo.probCold*cFInfo.initTime[location]-OffloadLatency)
//	}
//}
//
//func exponentialCDF(mean float64, x float64) float64 {
//	return 1 - math.Exp(-(1/mean)*(x))
//}
//
//// Used to modify the impact of the cost of cloud offloading
//var cost = 1.0 //0.0001
//var beta = 0.0 //1.0
//
//// TODO test
//var preference = 1.0
//
//func SolveProbabilities(m map[string]*functionInfo) {
//	if len(m) == 0 {
//		return
//	}
//
//	list := make([]cFInfoWithClass, 0)
//
//	objectiveFunctionEntries := make([]float64, 0)
//	memoryConstraintEntries := make([]golp.Entry, 0)
//	cpuConstraintEntries := make([]golp.Entry, 0)
//
//	classMap := make(map[string][]*classFunctionInfo)
//
//	functionPdIndex := make(map[string]int)
//
//	for _, fInfo := range m {
//		for class, cFInfo := range fInfo.invokingClasses {
//			list = append(list, cFInfoWithClass{cFInfo, class})
//
//			classFunctionList, prs := classMap[class]
//			if !prs {
//				classFunctionList = make([]*classFunctionInfo, 1)
//				classFunctionList[0] = cFInfo
//			} else {
//				classFunctionList = append(classFunctionList, cFInfo)
//			}
//
//			classMap[class] = classFunctionList
//		}
//
//		fInfo.probCold = fInfo.getProbCold(LOCAL)
//		fInfo.probColdOffload = fInfo.getProbCold(OFFLOADED)
//	}
//
//	numberOfFunctionClass = len(list)
//
//	//4 for every function
//	lp := golp.NewLP(0, numberOfFunctionClass*4)
//
//	for i := range list {
//		//Probability constraints
//		cFInfo := list[i]
//
//		if debug {
//			lp.SetColName(getPExecutionIndex(i), "PE"+list[i].name+list[i].class)
//			lp.SetColName(getPOffloadIndex(i), "PO"+list[i].name+list[i].class)
//			lp.SetColName(getPDropIndex(i), "PD"+list[i].name+list[i].class)
//			lp.SetColName(getShareIndex(i), "X"+list[i].name+list[i].class)
//		}
//
//		//Probability constraints
//		//TODO needed if the sum is < 1?
//		//lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), 1.0}}, golp.LE, 1)
//		//lp.AddConstraintSparse([]golp.Entry{{getPOffloadIndex(i), 1.0}}, golp.LE, 1)
//		//lp.AddConstraintSparse([]golp.Entry{{getPDropIndex(i), 1.0}}, golp.LE, 1)
//
//		//Sum of pe + pd + po = 1
//		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), 1.0},
//			{getPOffloadIndex(i), 1.0}, {getPDropIndex(i), 1.0}}, golp.EQ, 1)
//
//		//Constraint for the scaling value
//		//pe*time*arrival <= scale
//		lp.AddConstraintSparse([]golp.Entry{{getPExecutionIndex(i), cFInfo.meanDuration[LOCAL] * cFInfo.arrivals},
//			{getShareIndex(i), -1}}, golp.LE, 0)
//
//		objectiveFunctionEntries = append(objectiveFunctionEntries,
//			[]float64{cFInfo.arrivals * Classes[cFInfo.class].Utility *
//				getDeadlineSatisfactionProb(LOCAL, cFInfo, Classes[cFInfo.class].MaximumResponseTime),
//				cFInfo.arrivals*Classes[cFInfo.class].Utility*
//					getDeadlineSatisfactionProb(OFFLOADED, cFInfo, Classes[cFInfo.class].MaximumResponseTime) -
//					beta*cost*cFInfo.arrivals*cFInfo.meanDuration[OFFLOADED]*float64(cFInfo.memory),
//				0,
//				0}...)
//
//		memoryConstraintEntries = append(memoryConstraintEntries, []golp.Entry{{getShareIndex(i), float64(cFInfo.memory)}}...)
//
//		if cFInfo.cpu != 0 {
//			cpuConstraintEntries = append(cpuConstraintEntries, []golp.Entry{{getShareIndex(i), cFInfo.cpu}}...)
//		}
//
//		functionPdIndex[cFInfo.name+cFInfo.class] = getPDropIndex(i)
//	}
//
//	//Class constraint
//	for k, classList := range classMap {
//		//Remove some constraints
//		if Classes[k].CompletedPercentage == 0 {
//			continue
//		}
//
//		classConstraintEntries := make([]golp.Entry, 0)
//		arrivalSum := 0.0
//
//		for i := range classList {
//			classConstraintEntries =
//				append(classConstraintEntries, []golp.Entry{{functionPdIndex[classList[i].name+k], classList[i].arrivals}}...)
//
//			arrivalSum += classList[i].arrivals
//		}
//
//		lp.AddConstraintSparse(classConstraintEntries, golp.LE, (1-Classes[k].CompletedPercentage)*arrivalSum)
//	}
//
//	if len(memoryConstraintEntries) > 0 {
//		lp.AddConstraintSparse(memoryConstraintEntries, golp.LE, float64(node.Resources.MaxMemMB))
//	}
//
//	if len(cpuConstraintEntries) > 0 {
//		lp.AddConstraintSparse(cpuConstraintEntries, golp.LE, node.Resources.MaxCPUs)
//	}
//
//	//Objective function
//	lp.SetObjFn(objectiveFunctionEntries)
//	lp.SetMaximize()
//
//	start := time.Now()
//	sol := lp.Solve()
//	elapsed := time.Since(start)
//
//	vars := lp.Variables()
//
//	for i := range list {
//		cFInfo := list[i]
//
//		cFInfo.probExecute = vars[getPExecutionIndex(i)]
//		cFInfo.probOffload = vars[getPOffloadIndex(i)]
//		cFInfo.probDrop = vars[getPDropIndex(i)]
//		cFInfo.share = vars[getShareIndex(i)]
//	}
//
//	for name, index := range functionPColdMap {
//		_, prs := m[name]
//		if !prs {
//			continue
//		}
//
//		m[name].probCold = vars[index]
//	}
//
//	if debug {
//		log.Println(lp.WriteToString())
//		log.Printf("Resolution took %s", elapsed)
//		log.Println("Var: ", vars)
//		log.Println("Sol type: ", sol)
//		log.Println("Optimum: ", lp.Objective())
//	}
//}
