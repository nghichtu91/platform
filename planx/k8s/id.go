package k8s

var defaultId = Id()

type ResourceId struct {
	namespace string
	name      string
}

func Id() ResourceId {
	return ResourceId{}
}

func (id ResourceId) Namespace(ns string) ResourceId {
	id.namespace = ns
	return id
}

func (id ResourceId) Name(name string) ResourceId {
	id.name = name
	return id
}

// DefaultNsResource returns a resource identity with the default namespace.
func DefaultNsResource(resourceName string) ResourceId {
	return defaultId.Name(resourceName)
}

// IsValid if the resource identity is valid.
func (id ResourceId) IsValid() error {
	if id.namespace == "" {
		return ErrInvalidNamespace
	}
	if id.name == "" {
		return ErrInvalidResourceName
	}
	return nil
}
