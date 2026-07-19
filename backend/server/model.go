package server

import (
	"errors"
)

type ServiceResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ServiceCreateRequest struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type ServiceCreateResponse struct {
	TaskId string `json:"task_id"`
}

type ServiceUpdateRequest struct {
	Name  *string `json:"name,omitempty"`
	Image *string `json:"image,omitempty"`
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
