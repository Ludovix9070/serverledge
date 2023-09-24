package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
	"math"
	// "strconv"
	"strings"
)

// ChoiceNode receives one input and produces one result to one of two alternative nodes, based on condition
type ChoiceNode struct {
	Id           string
	BranchId     int
	input        map[string]interface{}
	Alternatives []DagNode
	Conditions   []Condition
	FirstMatch   int
}

func NewChoiceNode(conds []Condition) *ChoiceNode {
	return &ChoiceNode{
		Id:           shortuuid.New(),
		Conditions:   conds,
		Alternatives: make([]DagNode, len(conds)),
		FirstMatch:   -1,
	}
}

// The condition function must be created from the Dag specification in State Function Language or AFCL

func (c *ChoiceNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *ChoiceNode:
		c2 := cmp.(*ChoiceNode)
		if len(c.Conditions) != len(c2.Conditions) || len(c.Alternatives) != len(c2.Alternatives) {
			return false
		}
		for i := 0; i < len(c.Alternatives); i++ {
			if c.Alternatives[i] != c2.Alternatives[i] {
				return false
			}
			if !c.Conditions[i].Equals(c2.Conditions[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Exec for choice node evaluates the condition
func (c *ChoiceNode) Exec(*Progress) (map[string]interface{}, error) {
	// simply evalutes the Conditions and set the matching one
	for i, condition := range c.Conditions {
		ok, err := condition.Test()
		if err != nil {
			return nil, fmt.Errorf("error while testing condition: %v", err)
		}
		if ok {
			c.FirstMatch = i
			// the output map should be like the input map!
			return c.input, nil
		}
	}
	return nil, fmt.Errorf("no condition is met")
}

func (c *ChoiceNode) AddInput(dagNode DagNode) error {
	//if c.InputFrom != nil {
	//	return errors.New("input already present in node")
	//}
	//
	//c.InputFrom = dagNode
	return nil
}

// TODO: thats a bit useless
func (c *ChoiceNode) AddCondition(condition Condition) {
	c.Conditions = append(c.Conditions, condition)
}

func (c *ChoiceNode) AddOutput(dagNode DagNode) error {

	if len(c.Alternatives) > len(c.Conditions) {
		return errors.New(fmt.Sprintf("there are %d alternatives but %d Conditions", len(c.Alternatives), len(c.Conditions)))
	}
	c.Alternatives = append(c.Alternatives, dagNode)
	if len(c.Alternatives) > len(c.Conditions) {
		return errors.New(fmt.Sprintf("there are %d alternatives but %d Conditions", len(c.Alternatives), len(c.Conditions)))
	}
	return nil
}

func (c *ChoiceNode) ReceiveInput(input map[string]interface{}) error {
	c.input = input
	return nil
}

func (c *ChoiceNode) PrepareOutput(output map[string]interface{}) error {
	// we should map the output to the input of the node that first matches the condition and not to every alternative
	for _, n := range c.GetNext() {
		switch n.(type) {
		case *SimpleNode:
			return n.(*SimpleNode).MapOutput(output)
		}
	}
	return nil
}

// GetChoiceBranch returns all node ids of a branch under a choice node; branch number starts from 0
func (c *ChoiceNode) GetChoiceBranch(branch int) []DagNode {
	branchNodes := make([]DagNode, 0)
	if len(c.Alternatives) <= branch {
		fmt.Printf("fail to get branch %d\n", branch)
		return branchNodes
	}
	node := c.Alternatives[branch]
	return VisitDag(node, branchNodes, true)
}

func (c *ChoiceNode) GetNodesToSkip() []DagNode {
	nodesToSkip := make([]DagNode, 0)
	if c.FirstMatch == -1 || c.FirstMatch >= len(c.Alternatives) {
		return nodesToSkip
	}
	for i := 0; i < len(c.Alternatives); i++ {
		if i == c.FirstMatch {
			continue
		}
		nodesToSkip = append(nodesToSkip, c.GetChoiceBranch(i)...)
	}
	return nodesToSkip
}

func (c *ChoiceNode) GetNext() []DagNode {
	// you should have called exec before calling GetNext
	if c.FirstMatch >= len(c.Alternatives) {
		panic("there aren't sufficient alternatives!")
	}

	if c.FirstMatch < 0 {
		panic("first match cannot be less then 0. You should call Exec() before GetNext()")
	}
	next := make([]DagNode, 1)
	next[0] = c.Alternatives[c.FirstMatch]
	return next
}

func (c *ChoiceNode) Width() int {
	return len(c.Alternatives)
}

func (c *ChoiceNode) Name() string {
	n := len(c.Conditions)

	if n%2 == 0 {
		// se n =10 : -9 ---------
		// se n = 8 : -7 -------
		// se n = 6 : -5
		// se n = 4 : -3
		// se n = 2 : -1
		// [Simple|Simple|Simple|Simple|Simple|Simple|Simple|Simple|Simple|Simple]
		return strings.Repeat("-", 4*(n-1)-n/2) + "Choice" + strings.Repeat("-", 3*(n-1)+n/2)
	} else {
		pad := "-------"
		return strings.Repeat(pad, int(math.Max(float64(n/2), 0.))) + "Choice" + strings.Repeat(pad, int(math.Max(float64(n/2), 0.)))
	}
}

func (c *ChoiceNode) setBranchId(number int) {
	c.BranchId = number
}

func (c *ChoiceNode) GetBranchId() int {
	return c.BranchId
}

func (c *ChoiceNode) ToString() string {
	conditions := "<"
	for i, condFn := range c.Conditions {
		conditions += condFn.ToString()
		if i != len(c.Conditions) {
			conditions += " | "
		}
	}
	conditions += ">"
	return fmt.Sprintf("[ChoiceNode(%d): %s] ", c.Alternatives, conditions)
}

func (c *ChoiceNode) GetId() string {
	return c.Id
}
