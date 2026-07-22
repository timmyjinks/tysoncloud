package server

import (
	"errors"
	"time"
)

type VolumeResponse struct {
	Id        string    `json:"id"`
	ServiceId string    `json:"service_id"`
	MountPath string    `json:"mount_path"`
	StorageGB int32     `json:"storage_gb"`
	CreatedAt time.Time `json:"created_at"`
}

type VolumeCreateRequest struct {
	MountPath string `json:"mount_path"`
	StorageGB int32  `json:"storage_gb"`
}

type DatabaseResponse struct {
	Id             string    `json:"id"`
	ProjectId      string    `json:"project_id"`
	Name           string    `json:"name"`
	Engine         string    `json:"engine"`
	Port           int32     `json:"port"`
	Storage        int32     `json:"storage"`
	InternalDomain string    `json:"internal_domain"`
	CreatedAt      time.Time `json:"created_at"`
}

type DatabaseCreateRequest struct {
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	StorageGB int32  `json:"storage_gb"`
}

type DatabaseUpdateRequest struct {
	Name      *string `json:"name,omitempty"`
	Engine    *string `json:"engine,omitempty"`
	StorageGB *int32  `json:"storage_gb"`
}

type DatabaseDeleteRequest struct {
	Id string `json:"id"`
}

type ServiceResponse struct {
	Id             string    `json:"id"`
	ProjectId      string    `json:"project_id"`
	Name           string    `json:"name"`
	Image          string    `json:"image"`
	Port           int32     `json:"port"`
	Status         string    `json:"status"`
	PublicDomain   string    `json:"public_domain"`
	InternalDomain string    `json:"private_domain"`
	CreatedAt      time.Time `json:"created_at"`
}

type ServiceCreateRequest struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Port  int32  `json:"port"`
	Env   string `json:"env"`
}

type ServiceCreateResponse struct {
	TaskId string `json:"task_id"`
}

type ServiceUpdateRequest struct {
	Name  *string `json:"name,omitempty"`
	Image *string `json:"image,omitempty"`
	Port  *int32  `json:"port,omitempty"`
	Env   *string `json:"env"`
}

type ServiceDeleteRequest struct {
	Id string `json:"id"`
}

type ProjectResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ProjectCreateRequest struct {
	Name string `json:"name"`
}

type ProjectUpdateRequest struct {
	Name *string `json:"name,omitempty"`
}

var emptyName error = errors.New("name was empty")
var emptyImage error = errors.New("image was empty")
var invalidEnv error = errors.New("env was not valid KEY=VALUE lines")
