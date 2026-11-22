package deploy

type Service struct {
	Namespace string
	Name      string
	Hostname  string
	Env       map[string][]byte
	Image     string
	Port      int32
}

type Volume struct {
	MountPath string
	StorageGB int32
}

type Database struct {
	Namespace string
	Name      string
	Engine    string
	StorageGB int32
}
