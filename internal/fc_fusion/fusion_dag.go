package fc_fusion

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/grussorusso/serverledge/internal/cache"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"github.com/labstack/echo/v4"
)

// Evaluate fusion between nodes in the dag
func FuseFc(fcomp *fc.FunctionComposition) (bool, error) {
	PrintDag(&fcomp.Workflow)
	visited := map[fc.DagNodeId]bool{}
	nodeQueue := []fc.DagNodeId{fcomp.Workflow.Start.GetNext()[0]}
	changed := false

	/*err := cleanCurrentComp(fcomp)
	if err != nil {
		return false, err
	}*/

	for len(nodeQueue) > 0 {
		currentId := nodeQueue[0]
		nodeQueue = nodeQueue[1:]

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
						// Fusion
						func1, ok := function.GetFunction(simpleNode.Func)
						if !ok {
							return false, fmt.Errorf("fun1 error")
						}
						func2, ok := function.GetFunction(nextSimpleNode.Func)
						if !ok {
							return false, fmt.Errorf("fun2 error")
						}

						if containsPython(func1.Runtime) && containsPython(func2.Runtime) {
							if func1.Runtime != func2.Runtime {
								nodeQueue = append(nodeQueue, simpleNode.GetNext()...)
								continue
							}
						} else {
							nodeQueue = append(nodeQueue, simpleNode.GetNext()...)
							continue
						}

						mergedFunc, error := CombineFunctions(func1, func2)
						if error != nil {
							return false, fmt.Errorf("combining functions error")
						}

						//Update Functions structure of the composition with the new fused func
						fcomp.Functions[mergedFunc.Name] = mergedFunc
						newNodeId := fc.DagNodeId(fmt.Sprintf("%s_%s", simpleNode.Id, nextSimpleNode.Id))

						mergedNode := &fc.SimpleNode{
							Id:       newNodeId,
							NodeType: simpleNode.NodeType,
							BranchId: simpleNode.BranchId,
							OutputTo: nextSimpleNode.OutputTo,
							Func:     mergedFunc.Name,
						}

						// Adding new node to the dag
						fcomp.Workflow.Nodes[newNodeId] = mergedNode

						for _, prevNode := range fcomp.Workflow.Nodes {
							switch node := prevNode.(type) {
							case *fc.StartNode:
								if node.Next == simpleNode.Id || node.Next == nextSimpleNode.Id {
									//node.Next = newNodeId
									node.SetNext(newNodeId)
									fcomp.Workflow.Start = node
								}

							case *fc.SimpleNode:
								if node.OutputTo == simpleNode.Id || node.OutputTo == nextSimpleNode.Id {
									node.OutputTo = newNodeId
								}

							case *fc.FanOutNode:
								for i, next := range node.OutputTo {
									if next == simpleNode.Id || next == nextSimpleNode.Id {
										node.OutputTo[i] = newNodeId
									}
								}

							case *fc.FanInNode:
								if node.OutputTo == simpleNode.Id || node.OutputTo == nextSimpleNode.Id {
									node.OutputTo = newNodeId
								}

							case *fc.ChoiceNode:
								for i, next := range node.Alternatives {
									if next == simpleNode.Id || next == nextSimpleNode.Id {
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

						/*// Continue from the current node
						nodeQueue = append(nodeQueue, mergedNode.GetNext()...)*/

						// Re-evaluate the newly merged node for further fusions
						nodeQueue = append([]fc.DagNodeId{mergedNode.Id}, nodeQueue...)

						changed = true
						continue
					}
				}
			}

			//Add next node to visit queue
			nodeQueue = append(nodeQueue, simpleNode.GetNext()...)
		}
	}

	if changed {
		err := replaceWithNewFc(fcomp)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func IsFunctionUsed(funcName string, dag *fc.Dag) bool {
	for _, node := range dag.Nodes {
		if simpleNode, ok := node.(*fc.SimpleNode); ok {
			if simpleNode.Func == funcName {
				return true
			}
		}
	}
	return false
}

func replaceWithNewFc(comp *fc.FunctionComposition) error {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	dresp, err := cli.Delete(ctx, comp.GetEtcdKeyFromExt())
	if err != nil || dresp.Deleted != 1 {
		return fmt.Errorf("failed Delete: %v", err)
	}

	// Remove the function from the local cache
	cache.GetCacheInstance().Delete(comp.Name)

	err = comp.SaveToEtcd()
	if err != nil {
		log.Printf("Failed creation: %v", err)
		return fmt.Errorf("failed new fc creation")
	}

	return nil
}

/*func cleanCurrentComp(comp *fc.FunctionComposition) error {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	dresp, err := cli.Delete(ctx, comp.GetEtcdKeyFromExt())
	if err != nil || dresp.Deleted != 1 {
		return fmt.Errorf("failed Delete: %v", err)
	}

	// Remove the function from the local cache
	cache.GetCacheInstance().Delete(comp.Name)

	return nil
}

func replaceWithNewFc(comp *fc.FunctionComposition) error {
	err := comp.SaveToEtcd()
	if err != nil {
		log.Printf("Failed creation: %v", err)
		return fmt.Errorf("failed new fc creation")
	}

	//ONLY FOR DEBUG
	//only to check if etcd accessed in the subsequent invocation
	//cache.GetCacheInstance().Delete(comp.Name)

	return nil
}*/

func containsPython(s string) bool {
	// Verifica se la stringa contiene "python" o "Python"
	return strings.Contains(s, "python") || strings.Contains(s, "Python")
}

// ONLY FOR DEBUG
func PrintDag(dag *fc.Dag) {
	fmt.Println("DAG State:")
	for id, node := range dag.Nodes {
		fmt.Printf("Node ID: %s, Type: %T\n", id, node)
		if simpleNode, ok := node.(*fc.SimpleNode); ok {
			fmt.Printf("  Func: %s, OutputTo: %v\n", simpleNode.Func, simpleNode.OutputTo)
		}
		if hasNext, ok := node.(fc.HasNext); ok {
			fmt.Printf("  Next: %v\n", hasNext.GetNext())
		}
	}
}

func PrintDagSimpleJSON(c echo.Context, dag *fc.Dag) error {
	type NodeJSON struct {
		NodeID   string   `json:"node_id"`
		NodeType string   `json:"node_type"`
		Func     string   `json:"func,omitempty"`
		OutputTo []string `json:"output_to,omitempty"`
		Next     []string `json:"next,omitempty"`
	}

	var jsonOutput []NodeJSON

	for id, node := range dag.Nodes {
		nodeJSON := NodeJSON{
			NodeID:   string(id),
			NodeType: fmt.Sprintf("%T", node), // node type
		}

		if simpleNode, ok := node.(*fc.SimpleNode); ok {
			nodeJSON.Func = simpleNode.Func

			nodeJSON.OutputTo = []string{string(simpleNode.OutputTo)}
		}

		if hasNext, ok := node.(fc.HasNext); ok {

			for _, nextID := range hasNext.GetNext() {
				nodeJSON.Next = append(nodeJSON.Next, string(nextID))
			}
		}

		jsonOutput = append(jsonOutput, nodeJSON)
	}

	return c.JSON(http.StatusOK, jsonOutput)
}

func PrintDagOrderedJSON(c echo.Context, dag *fc.Dag) error {
	type NodeJSON struct {
		NodeID   string   `json:"node_id"`
		NodeType string   `json:"node_type"`
		Func     string   `json:"func,omitempty"`
		OutputTo []string `json:"output_to,omitempty"`
		Next     []string `json:"next,omitempty"`
	}

	// Map to track dependancies
	adjacencyMap := make(map[string][]string)
	inDegree := make(map[string]int)

	// Costructs the adjacency map
	for id, node := range dag.Nodes {
		if hasNext, ok := node.(fc.HasNext); ok {
			adjacencyMap[string(id)] = []string{}
			for _, nextID := range hasNext.GetNext() {
				adjacencyMap[string(id)] = append(adjacencyMap[string(id)], string(nextID))
				inDegree[string(nextID)]++
			}
		}
	}

	// Nodes without initial dependancies
	var order []string
	queue := []string{}

	for id := range dag.Nodes {
		if inDegree[string(id)] == 0 {
			queue = append(queue, string(id))
		}
	}

	// Topological ordering using BFS (Kahn's Algorithm)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, neighbor := range adjacencyMap[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Creation of the ordered Json output
	var jsonOutput []NodeJSON
	for _, id := range order {
		nodeID := fc.DagNodeId(id)
		node := dag.Nodes[nodeID]

		nodeJSON := NodeJSON{
			NodeID:   id,
			NodeType: fmt.Sprintf("%T", node),
		}

		if simpleNode, ok := node.(*fc.SimpleNode); ok {
			nodeJSON.Func = simpleNode.Func
			nodeJSON.OutputTo = []string{string(simpleNode.OutputTo)}
		}

		if hasNext, ok := node.(fc.HasNext); ok {
			for _, nextID := range hasNext.GetNext() {
				nodeJSON.Next = append(nodeJSON.Next, string(nextID))
			}
		}

		jsonOutput = append(jsonOutput, nodeJSON)
	}

	return c.JSON(http.StatusOK, jsonOutput)
}
