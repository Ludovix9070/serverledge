package fc_fusion

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"

	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/lithammer/shortuuid"
)

// Evaluate fusion between nodes in the dag
func FuseFcEvaluate(fcomp *fc.FunctionComposition, policyDef policyDefinitionTerms, evalFunctions []functionElem) (bool, error) {
	PrintDag(&fcomp.Workflow)
	visited := map[fc.DagNodeId]bool{}
	//nodeQueue := []fc.DagNodeId{fcomp.Workflow.Start.GetNext()[0]}
	nodeQueue := []QueueNode{{NodeID: fcomp.Workflow.Start.GetNext()[0], IsFusible: false, Evaluated: false}}

	changed := false

	//err := cleanCurrentComp(fcomp)
	//if err != nil {
	//return false, err
	//}

	for len(nodeQueue) > 0 {
		current := nodeQueue[0]
		nodeQueue = nodeQueue[1:]

		currentId := current.NodeID
		fmt.Println("DagNodeId corrente ", currentId)

		if visited[currentId] {
			continue
		}
		visited[currentId] = true

		currentNode, ok := fcomp.Workflow.Find(currentId)
		if !ok {
			continue
		}

		// Check if it's a simple node
		if simpleNode, ok := currentNode.(*fc.SimpleNode); ok {
			nextNodes := simpleNode.GetNext()
			if len(nextNodes) == 1 {
				nextNodeId := nextNodes[0]
				nextNode, ok := fcomp.Workflow.Find(nextNodeId)
				if ok {
					// Check if next node is a simple node
					if nextSimpleNode, ok := nextNode.(*fc.SimpleNode); ok {
						//controlli necessari per valutare se fusione effettivamente ok
						//posso essere stato già valutato se vengo da choice o fanout
						//o se sono il next di un simple node candidato alla fusione
						if !current.Evaluated {
							current.Evaluated = true
							if CountNodeReferences(simpleNode.Id, &fcomp.Workflow) > 1 {
								current.IsFusible = false
								for _, nextNodeId := range simpleNode.GetNext() {
									nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
								}
								continue
							} else {
								current.IsFusible = true
								if CountNodeReferences(nextSimpleNode.Id, &fcomp.Workflow) > 1 {
									for _, nextNodeId := range simpleNode.GetNext() {
										nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: true})
									}
									continue
								}
							}

						} else if !current.IsFusible {
							for _, nextNodeId := range simpleNode.GetNext() {
								nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
							}
							continue
						} else {
							if CountNodeReferences(nextSimpleNode.Id, &fcomp.Workflow) > 1 {
								for _, nextNodeId := range simpleNode.GetNext() {
									nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: true})
								}
								continue
							}
						}

						// Fusion
						func1, ok := function.GetFunction(simpleNode.Func)
						if !ok {
							return false, fmt.Errorf("fun1 error")
						}

						canBeFused1, _ := containsFunElem(evalFunctions, func1.Name)
						/*if !found {
							return false, fmt.Errorf("fun1 found error between functions' array")
						}*/
						if !canBeFused1 {
							fmt.Printf("Function %s not fusible for the current policy\n", func1.Name)
							for _, nextNodeId := range simpleNode.GetNext() {
								nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
							}
							continue
						}

						func2, ok := function.GetFunction(nextSimpleNode.Func)
						if !ok {
							return false, fmt.Errorf("fun2 error")
						}

						canBeFused2, _ := containsFunElem(evalFunctions, func2.Name)
						/*if !found {
							return false, fmt.Errorf("fun1 found error between functions' array")
						}*/

						if !canBeFused2 {
							fmt.Printf("Function %s not fusible for the current policy\n", func2.Name)
							for _, nextNodeId := range simpleNode.GetNext() {
								nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
							}
							continue
						}

						if policyDef.MaxCpuDelta.isAct {
							cpuDelta := math.Abs(func1.CPUDemand - func2.CPUDemand)
							if cpuDelta > policyDef.MaxCpuDelta.threshold {
								fmt.Printf("  CpuDelta %f supera la soglia di CpuDemand %f. Le due funzioni non possono essere fuse.\n", cpuDelta, policyDef.MaxCpuDelta.threshold)
								for _, nextNodeId := range simpleNode.GetNext() {
									nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
								}
								continue
							} else {
								fmt.Println("Passed Max Cpu Delta Policy Check")
							}
						}

						if policyDef.MaxMemoryDelta.isAct {
							memDelta := math.Abs(float64(func1.MemoryMB) - float64(func2.MemoryMB))
							if memDelta > policyDef.MaxMemoryDelta.threshold {
								fmt.Printf("  MemDelta %f supera la soglia di MaxMemDelta %f. Le due funzioni non possono essere fuse.\n", memDelta, policyDef.MaxMemoryDelta.threshold)
								for _, nextNodeId := range simpleNode.GetNext() {
									nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
								}
								continue
							} else {
								fmt.Println("Passed Max Memory Delta Policy Check")
							}
						}

						if policyDef.MaxDimPkt.isAct {
							tarBytesFunc1, _ := base64.StdEncoding.DecodeString(func1.TarFunctionCode)
							tarBytesFunc2, _ := base64.StdEncoding.DecodeString(func2.TarFunctionCode)
							maxApproxDim := bytesToMB(len(tarBytesFunc1) + len(tarBytesFunc2))
							fmt.Printf("Dim Combinata Pkt %f\n", maxApproxDim)
							if maxApproxDim > policyDef.MaxDimPkt.threshold {
								fmt.Printf("  Dim Pkt Finale %f supera la soglia di MaxDimPkt %f. Le due funzioni non possono essere fuse.\n", maxApproxDim, policyDef.MaxDimPkt.threshold)
								for _, nextNodeId := range simpleNode.GetNext() {
									nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
								}
								continue
							} else {
								fmt.Println("Passed Max Dim Pkt Policy Check")
							}
						}

						if containsPython(func1.Runtime) && containsPython(func2.Runtime) {
							if func1.Runtime != func2.Runtime {
								for _, nextNodeId := range simpleNode.GetNext() {
									nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
								}
								continue
							}
						} else {
							for _, nextNodeId := range simpleNode.GetNext() {
								nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
							}
							continue
						}

						mergedFunc, error := CombineFunctions(func1, func2)
						fmt.Println("Primo giro")
						if error != nil {
							fmt.Println("Giro con errore ", error)
							return false, fmt.Errorf("combining functions error")
						}

						//Update Functions structure of the composition with the new fused func
						fcomp.Functions[mergedFunc.Name] = mergedFunc
						//per evitare esplosioni, si può usare la versione usata per un normale simple node invece che per uno proveniente da asl
						//newNodeId := fc.DagNodeId(fmt.Sprintf("%s_%s", simpleNode.Id, nextSimpleNode.Id))
						newNodeId := fc.DagNodeId(shortuuid.New())

						mergedNode := &fc.SimpleNode{
							Id:       newNodeId,
							NodeType: simpleNode.NodeType,
							BranchId: simpleNode.BranchId,
							OutputTo: nextSimpleNode.OutputTo,
							Func:     mergedFunc.Name,
						}

						// Adding new node to the dag
						fcomp.Workflow.Nodes[newNodeId] = mergedNode

						//qui aggiorno i riferimenti di tutti i nodi che riferiscono il primo nodo
						for _, prevNode := range fcomp.Workflow.Nodes {
							switch node := prevNode.(type) {
							case *fc.StartNode:
								//if node.Next == simpleNode.Id || node.Next == nextSimpleNode.Id {
								if node.Next == simpleNode.Id {
									//node.Next = newNodeId
									node.SetNext(newNodeId)
									fcomp.Workflow.Start = node
								}

							case *fc.SimpleNode:
								//if node.OutputTo == simpleNode.Id || node.OutputTo == nextSimpleNode.Id {
								if node.OutputTo == simpleNode.Id {
									node.OutputTo = newNodeId
								}

							case *fc.FanOutNode:
								for i, next := range node.OutputTo {
									//if next == simpleNode.Id || next == nextSimpleNode.Id {
									if next == simpleNode.Id {
										node.OutputTo[i] = newNodeId
									}
								}

							case *fc.FanInNode:
								//if node.OutputTo == simpleNode.Id || node.OutputTo == nextSimpleNode.Id {
								if node.OutputTo == simpleNode.Id {
									node.OutputTo = newNodeId
								}

							case *fc.ChoiceNode:
								for i, next := range node.Alternatives {
									//if next == simpleNode.Id || next == nextSimpleNode.Id {
									if next == simpleNode.Id {
										node.Alternatives[i] = newNodeId
									}
								}

							default:
								log.Printf("Unhandled node type: %T", node)
							}
						}

						// Remove old nodes
						delete(fcomp.Workflow.Nodes, simpleNode.Id)
						delete(fcomp.Workflow.Nodes, nextSimpleNode.Id)

						// Removing functions use by fused node if not used in another node in the dag
						if !IsFunctionUsed(func1.Name, &fcomp.Workflow) {
							delete(fcomp.Functions, func1.Name)
						}
						if !IsFunctionUsed(func2.Name, &fcomp.Workflow) {
							delete(fcomp.Functions, func2.Name)
						}

						// Continue from the current node
						//nodeQueue = append(nodeQueue, mergedNode.GetNext()...)

						// Re-evaluate the newly merged node for further fusions
						nodeQueue = append(nodeQueue, QueueNode{NodeID: mergedNode.Id, IsFusible: false, Evaluated: false})

						changed = true
						continue
					}
				}
			}

			//Add next node to visit queue
			for _, nextNodeId := range simpleNode.GetNext() {
				nodeQueue = append(nodeQueue, QueueNode{NodeID: nextNodeId, IsFusible: false, Evaluated: false})
			}
		} else {
			switch node := currentNode.(type) {
			case *fc.ChoiceNode:
				// Aggiungi tutte le alternative alla coda
				for _, alt := range node.Alternatives {
					nextNode, ok := fcomp.Workflow.Find(alt)
					if ok {
						if nextSimpleNode, ok := nextNode.(*fc.SimpleNode); ok {
							fusible := CanFuseChoiceNode(nextSimpleNode, &fcomp.Workflow, node)
							nodeQueue = append(nodeQueue, QueueNode{NodeID: alt, IsFusible: fusible, Evaluated: true})
						} else {
							//non è un simple node, lo valuto di default come non fusible
							nodeQueue = append(nodeQueue, QueueNode{NodeID: alt, IsFusible: false, Evaluated: true})
						}
					}
				}

			case *fc.FanOutNode:
				for _, branch := range node.OutputTo {
					nextNode, ok := fcomp.Workflow.Find(branch)
					if ok {
						if nextSimpleNode, ok := nextNode.(*fc.SimpleNode); ok {
							fusible := CanFuseFanOutNode(nextSimpleNode, &fcomp.Workflow, node)
							nodeQueue = append(nodeQueue, QueueNode{NodeID: branch, IsFusible: fusible, Evaluated: true})
						} else {
							nodeQueue = append(nodeQueue, QueueNode{NodeID: branch, IsFusible: false, Evaluated: true})
						}
					}
				}

			case *fc.FanInNode:
				// Aggiungi il nodo successivo alla coda
				if node.OutputTo != "" {
					nodeQueue = append(nodeQueue, QueueNode{NodeID: node.OutputTo, IsFusible: false, Evaluated: false})
				}

			default:
				log.Printf("Unhandled node type: %T", node)
			}
		}
	}

	if changed {
		err := replaceWithNewFc(fcomp)
		if err != nil {
			return false, err
		}
	}

	fmt.Println("DIM FINALE ", len(fcomp.Workflow.Nodes))

	return true, nil
}

func containsFunElem(slice []functionElem, str string) (bool, bool) {
	for _, item := range slice {
		if item.name == str {
			return item.canBeFused, true
		}
	}

	//se non ho trovato informazioni pregresse, significa che è un nodo già fuso, per ora lo marco come fusibile
	return true, false
}

func bytesToMB(bytes int) float64 {
	return float64(bytes) / (1024 * 1024) // Dividi per 1.048.576
}
