package server

import (
	"encoding/json"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
)

func (app *Application) GetVolume(w http.ResponseWriter, r *http.Request) {
	serviceId := mux.Vars(r)["service_id"]
	if serviceId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	volume, err := app.Supabase.GetVolume(serviceId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(VolumeResponse{Id: volume.Id, ServiceId: volume.ServiceId, MountPath: volume.MountPath, StorageGB: volume.StorageGB}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) CreateVolume(w http.ResponseWriter, r *http.Request) {
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

	var service VolumeCreateRequest
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

	if _, err := app.Supabase.CreateVolume(serviceId, userId, service.MountPath, service.StorageGB); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.AttachVolume(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      "svc-" + serviceId,
	}, deploy.Volume{
		MountPath: service.MountPath,
		StorageGB: service.StorageGB,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) DeleteVolume(w http.ResponseWriter, r *http.Request) {
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

	if err := app.Supabase.DeleteVolume(serviceId, userId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.DetachVolume(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      "svc-" + serviceId,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(204)
}
