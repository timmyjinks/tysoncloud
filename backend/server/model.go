package server

type ServiceCreateRequest struct {
	Name   string `json:"name"`
	Volume string `json:"volume"`
	Type   string `json:"type"`
}

type ServiceCreateResponse struct {
	TaskId string `json:"task_id"`
}

type ServiceUpdateRequest struct {
	Id      string `json:"id"`
	Newname string `json:"name"`
	Env     string `json:"env"`
	Volume  string `json:"volume"`
}

type ServiceDeleteRequest struct {
	Id string `json:"id"`
}

type ProjectCreateRequest struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Env    string `json:"env"`
	Volume string `json:"volume"`
}

type ProjectUpdateRequest struct {
	Id      string `json:"id"`
	Newname string `json:"name"`
	Env     string `json:"env"`
	Volume  string `json:"volume"`
}

type ProjectDeleteRequest struct {
	Id string `json:"id"`
}
