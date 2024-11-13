package fc_fusion

type FusionPolicy interface {
	Init()
	OnCompletion(info *fusionInfo)
	OnArrival(info *fusionInfo)
}
