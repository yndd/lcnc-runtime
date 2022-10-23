package fnruntime

type ImagePullPolicy string

const (
	AlwaysPull       ImagePullPolicy = "Always"
	IfNotPresentPull ImagePullPolicy = "IfNotPresent"
	NeverPull        ImagePullPolicy = "Never"
)

/*
var allImagePullPolicy = []ImagePullPolicy{
	AlwaysPull,
	IfNotPresentPull,
	NeverPull,
}
*/
