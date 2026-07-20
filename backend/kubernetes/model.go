package kubernetes

type Resource struct {
	Namespace string
	Name      string
	Hostname  string
	Env       map[string][]byte
	Image     string
	Port      int32
	Engine    string
	StorageGB int32
	MountPath string
}
