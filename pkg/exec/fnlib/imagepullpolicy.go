package fnlib

type ImagePullPolicy string

const (
	AlwaysPull       ImagePullPolicy = "Always"
	IfNotPresentPull ImagePullPolicy = "IfNotPresent"
	NeverPull        ImagePullPolicy = "Never"
)
