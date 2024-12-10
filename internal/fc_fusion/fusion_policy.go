package fc_fusion

type FusionPolicy interface {
	Init()
	OnCompletion(fr *fusionRequest)
	OnArrival(fr *fusionRequest)
}
