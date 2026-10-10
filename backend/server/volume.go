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
		writeError(w, http.StatusBadRequest, "A service ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	if _, err := app.Supabase.GetService(serviceId, claims.Subject); err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	volume, err := app.Supabase.GetVolume(serviceId)
	if err != nil {
		writeError(w, http.StatusNotFound, "This service doesn't have a volume attached.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(VolumeResponse{Id: volume.Id, ServiceId: volume.ServiceId, MountPath: volume.MountPath, StorageGB: volume.StorageGB}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) CreateVolume(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	serviceId := mux.Vars(r)["service_id"]
	if serviceId == "" {
		writeError(w, http.StatusBadRequest, "A service ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var volume VolumeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&volume); err != nil {
		writeError(w, http.StatusBadRequest, "That volume request wasn't valid.", err)
		return
	}

	if volume.MountPath == "" {
		writeError(w, http.StatusBadRequest, "A mount path is required.", nil)
		return
	}
	if volume.StorageGB <= 0 {
		writeError(w, http.StatusBadRequest, "Storage size must be greater than zero.", nil)
		return
	}

	userId := claims.Subject

	service, err := app.Supabase.GetService(serviceId, userId)
	if err != nil || service.ProjectId != projectId {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	if _, err := app.Supabase.CreateVolume(serviceId, userId, volume.MountPath, volume.StorageGB); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't attach the volume.", err)
		return
	}

	if err := app.Deploy.AttachVolume(r.Context(), deploy.Service{
		Namespace: "proj-" + service.ProjectId,
		Name:      "svc-" + serviceId,
	}, deploy.Volume{
		MountPath: volume.MountPath,
		StorageGB: volume.StorageGB,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Your volume was created, but we couldn't attach it. A refresh will show its current status.", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	serviceId := mux.Vars(r)["service_id"]
	if serviceId == "" {
		writeError(w, http.StatusBadRequest, "A service ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	service, err := app.Supabase.GetService(serviceId, claims.Subject)
	if err != nil || service.ProjectId != projectId {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	if err := app.Supabase.DeleteVolume(serviceId, claims.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't detach the volume.", err)
		return
	}

	if err := app.Deploy.DetachVolume(r.Context(), deploy.Service{
		Namespace: "proj-" + service.ProjectId,
		Name:      "svc-" + serviceId,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Your volume was removed, but we couldn't detach it from your running service. A refresh will show its current status.", err)
		return
	}

	w.WriteHeader(204)
}
