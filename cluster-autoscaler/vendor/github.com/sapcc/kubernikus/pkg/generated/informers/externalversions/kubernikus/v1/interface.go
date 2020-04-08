// This file was automatically generated by informer-gen

package v1

import (
	internalinterfaces "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Klusters returns a KlusterInformer.
	Klusters() KlusterInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Klusters returns a KlusterInformer.
func (v *version) Klusters() KlusterInformer {
	return &klusterInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
