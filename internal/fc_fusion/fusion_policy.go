package fc_fusion

type FusionPolicy interface {
	Init()
	OnCompletion(info ReturnedOutputData)
	OnArrival(info ReturnedOutputData)
}
