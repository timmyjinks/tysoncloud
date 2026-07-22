package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
)

var invalidServiceId error = errors.New("service with id not found")

func (app *Application) GetService(w http.ResponseWriter, r *http.Request) {
	serviceId := mux.Vars(r)["service_id"]
	if serviceId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	service, err := app.Supabase.GetService(serviceId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ServiceResponse{
		Id:             service.Id,
		ProjectId:      service.ProjectId,
		Name:           service.Name,
		Image:          service.Image,
		Port:           service.Port,
		Status:         service.Status,
		PublicDomain:   service.PublicDomain,
		InternalDomain: service.PrivateDomain,
		CreatedAt:      service.CreatedAt,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) GetServices(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	services, err := app.Supabase.GetServices(projectId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToServicesResponse(services)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) CreateService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]

	var service ServiceCreateRequest
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	res, err := app.Supabase.CreateService(userId, projectId, service.Name, service.Image, service.Port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       map[string][]byte{},
		Image:     service.Image,
		Port:      service.Port,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// if err := app.Cloudflare.CreateRecord(r.Context(), res.PublicDomain); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	//
	// if err := app.Cloudflare.CreateRoute(r.Context(), res.PublicDomain); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) UpdateService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	serviceId := mux.Vars(r)["service_id"]
	if serviceId == "" {
		http.Error(w, invalidServiceId.Error(), http.StatusBadRequest)
		return
	}

	var service ServiceUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	if service.Name == nil {
		http.Error(w, emptyName.Error(), http.StatusBadRequest)
		return
	}

	if service.Image == nil {
		http.Error(w, emptyImage.Error(), http.StatusBadRequest)
		return
	}

	if service.Port == nil {
		http.Error(w, emptyImage.Error(), http.StatusBadRequest)
		return
	}

	res, err := app.Supabase.UpdateService(serviceId, userId, *service.Name, *service.Image, *service.Port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       map[string][]byte{},
		Image:     *service.Image,
		Port:      *service.Port,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

}

func (app *Application) DeleteService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	serviceId := mux.Vars(r)["service_id"]
	if serviceId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	if err := app.Supabase.DeleteService(serviceId, userId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      "svc-" + serviceId,
	}); err != nil {
		return
	}

	// if err := app.Cloudflare.DeleteRecord(r.Context(), "tc-"+serviceId); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	//
	// if err := app.Cloudflare.DeleteRoute(r.Context(), "tc-"+serviceId); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	w.WriteHeader(204)
}

func ToServicesResponse(servicesTable []store.ServicesTable) []ServiceResponse {
	var services []ServiceResponse = []ServiceResponse{}
	for _, service := range servicesTable {
		services = append(services, ServiceResponse{
			Id:             service.Id,
			ProjectId:      service.ProjectId,
			Name:           service.Name,
			Image:          service.Image,
			Port:           service.Port,
			Status:         service.Status,
			PublicDomain:   service.PublicDomain,
			InternalDomain: service.PrivateDomain,
			CreatedAt:      service.CreatedAt,
		})
	}
	return services
}
