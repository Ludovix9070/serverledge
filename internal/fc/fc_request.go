package fc

import (
	"context"
	"fmt"
	"time"

	"github.com/cornelk/hashmap"
	"github.com/grussorusso/serverledge/internal/function"
)

type ReqId string

// CompositionRequest represents a single function composition internal invocation, with params and metrics data
type CompositionRequest struct {
	Ctx             context.Context
	Fc              *FunctionComposition
	Params          map[string]interface{}
	Arrival         time.Time
	ExecReport      CompositionExecutionReport     // each function has its execution report, and the composition has additional metrics
	RequestQoSMap   map[string]function.RequestQoS // every function should have its RequestQoS
	CanDoOffloading bool                           // every function inherits this flag
	Async           bool
}

func NewCompositionRequest(ctx context.Context, composition *FunctionComposition, params map[string]interface{}) *CompositionRequest {
	return &CompositionRequest{
		Ctx:     ctx,
		Fc:      composition,
		Params:  params,
		Arrival: time.Now(),
		ExecReport: CompositionExecutionReport{
			Reports: hashmap.New[ExecutionReportId, *function.ExecutionReport](), // make(map[ExecutionReportId]*function.ExecutionReport),
		},
		RequestQoSMap:   make(map[string]function.RequestQoS),
		CanDoOffloading: true,
		Async:           false,
	}
}

func (r *CompositionRequest) Id() string {
	return r.Ctx.Value("ReqId").(string)
}

func (r *CompositionRequest) String() string {
	return fmt.Sprintf("[%s] Rq-%s", r.Fc.Name, r.Id())
}

type CompositionResponse struct {
	Success      bool
	Result       map[string]interface{}
	Reports      map[string]*function.ExecutionReport
	ResponseTime float64 // time waited by the user to get the output of the entire composition (in seconds)
}

type CompositionAsyncResponse struct {
	ReqId string
}
