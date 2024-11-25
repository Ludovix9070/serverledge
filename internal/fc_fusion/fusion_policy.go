package fc_fusion

type FusionPolicy interface {
	Init()
	OnCompletion(info *ReturnedOutputData, fr *fusionRequest)
	OnArrival(info *ReturnedOutputData, fr *fusionRequest)
}
