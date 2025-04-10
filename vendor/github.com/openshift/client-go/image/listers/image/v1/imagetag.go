// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	imagev1 "github.com/openshift/api/image/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// ImageTagLister helps list ImageTags.
// All objects returned here must be treated as read-only.
type ImageTagLister interface {
	// List lists all ImageTags in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*imagev1.ImageTag, err error)
	// ImageTags returns an object that can list and get ImageTags.
	ImageTags(namespace string) ImageTagNamespaceLister
	ImageTagListerExpansion
}

// imageTagLister implements the ImageTagLister interface.
type imageTagLister struct {
	listers.ResourceIndexer[*imagev1.ImageTag]
}

// NewImageTagLister returns a new ImageTagLister.
func NewImageTagLister(indexer cache.Indexer) ImageTagLister {
	return &imageTagLister{listers.New[*imagev1.ImageTag](indexer, imagev1.Resource("imagetag"))}
}

// ImageTags returns an object that can list and get ImageTags.
func (s *imageTagLister) ImageTags(namespace string) ImageTagNamespaceLister {
	return imageTagNamespaceLister{listers.NewNamespaced[*imagev1.ImageTag](s.ResourceIndexer, namespace)}
}

// ImageTagNamespaceLister helps list and get ImageTags.
// All objects returned here must be treated as read-only.
type ImageTagNamespaceLister interface {
	// List lists all ImageTags in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*imagev1.ImageTag, err error)
	// Get retrieves the ImageTag from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*imagev1.ImageTag, error)
	ImageTagNamespaceListerExpansion
}

// imageTagNamespaceLister implements the ImageTagNamespaceLister
// interface.
type imageTagNamespaceLister struct {
	listers.ResourceIndexer[*imagev1.ImageTag]
}
