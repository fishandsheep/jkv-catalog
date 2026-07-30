package provider

import "context"

// Discoverer resolves reviewed source coordinates into releases for one platform.
type Discoverer interface {
	Discover(context.Context, Platform) ([]Discovery, error)
}

// Default returns discovery adapter for a supported candidate.
func Default(candidate string) (Discoverer, bool) {
	if source, ok := DefaultFlatArchive(candidate); ok {
		return source, true
	}
	if candidate == "maven" {
		return DefaultMaven(), true
	}
	switch candidate {
	case "java":
		return Java{}, true
	case "groovy":
		return DefaultGroovy(), true
	case "tomcat":
		return DefaultTomcat(), true
	case "springboot":
		return DefaultSpringBoot(), true
	}
	return nil, false
}
