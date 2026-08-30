package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

func (app *Application) GetService(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	env, err := app.Deploy.GetServiceEnv(r.Context(), deploy.Service{
		Namespace: "proj-" + service.ProjectId,
		Name:      service.ResourceName,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the service's environment variables.", err)
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
		Env:            env,
		CreatedAt:      service.CreatedAt,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) GetServices(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	services, err := app.Supabase.GetServices(projectId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the project's services.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToServicesResponse(services)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) GetServiceLogs(w http.ResponseWriter, r *http.Request) {
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

	token := r.URL.Query().Get("token")
	if token == "" {
		cookie, cookieErr := r.Cookie("__session")
		if cookieErr != nil {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, cookieErr)
			return
		}
		token = cookie.Value
	}

	claims, err := clerkjwt.Verify(r.Context(), &clerkjwt.VerifyParams{Token: token})
	if err != nil {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, err)
		return
	}

	service, err := app.Supabase.GetService(serviceId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	allowedOrigins := parseAllowedOrigins(app.Config.Server.AllowedOrigins)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			return allowedOrigins[origin]
		},
	}
	ws, err := upgrader.Upgrade(w, r, http.Header{})
	if err != nil {
		slog.Error("log stream upgrade failed", "err", err)
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	lines := make(chan string)
	go func() {
		defer close(lines)
		svc := deploy.Service{
			Namespace: "proj-" + projectId,
			Name:      service.ResourceName,
		}
		status := strings.ToLower(strings.TrimSpace(service.Status))
		isDiagnostic := status == "pending" || status == "failed"
		var logErr error
		if isDiagnostic {
			logErr = app.Deploy.GetServiceDiagnosticLogs(ctx, svc, lines)
		} else {
			logErr = app.Deploy.GetServiceLogs(ctx, svc, lines)
		}
		if logErr != nil && ctx.Err() == nil {
			slog.Error("log stream failed", "service_id", serviceId, "status", service.Status, "err", logErr)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ws.WriteJSON(struct {
				Message string `json:"message"`
			}{Message: line}); err != nil {
				cancel()
				return
			}
		}
	}
}

func (app *Application) CreateService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var service ServiceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		writeError(w, http.StatusBadRequest, "That service request wasn't valid.", err)
		return
	}

	if service.Name == "" {
		writeError(w, http.StatusBadRequest, "Service name is required.", nil)
		return
	}
	if service.Image == "" {
		writeError(w, http.StatusBadRequest, "A Docker image is required.", nil)
		return
	}
	if ok, err := util.ValidateEnv(service.Env); err != nil || !ok {
		writeError(w, http.StatusBadRequest, "Environment variables must be one KEY=value pair per line.", err)
		return
	}
	if service.Domain != nil {
		normalized := util.NormalizeDomain(*service.Domain)
		if normalized == "" {
			service.Domain = nil
		} else {
			if ok, _ := util.ValidateDomainLabel(normalized); !ok {
				writeError(w, http.StatusBadRequest, "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen.", nil)
				return
			}
			service.Domain = &normalized
		}
	}

	userId := claims.Subject

	domainRequested := service.Domain != nil

	res, err := app.Supabase.CreateService(userId, projectId, service.Name, service.Image, service.Domain, service.Port)
	if err != nil {
		if domainRequested && isDomainTakenError(err) {
			writeError(w, http.StatusConflict, "That domain is already taken. Please choose a different one.", err)
			return
		}
		if domainRequested && isDomainValidationError(err) {
			writeError(w, http.StatusBadRequest, domainValidationMessage(err), err)
			return
		}
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "That name is already taken. Please choose a different one.", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Couldn't create the service.", err)
		return
	}

	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       util.ParseEnv(service.Env),
		Image:     service.Image,
		Port:      service.Port,
	}); err != nil {
		if _, statusErr := app.Supabase.UpdateServiceStatus(res.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark service failed after deploy error", "service_id", res.Id, "err", statusErr)
		}
		writeError(w, http.StatusInternalServerError, "Your service was created, but we couldn't start it. A refresh will show its current status.", err)
		return
	}

	if _, err := app.Supabase.UpdateServiceStatus(res.Id, userId, "running"); err != nil {
		writeError(w, http.StatusInternalServerError, "Your service was started, but we couldn't confirm its status. A refresh will show where things stand.", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) UpdateService(w http.ResponseWriter, r *http.Request) {
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

	var service ServiceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		writeError(w, http.StatusBadRequest, "That service request wasn't valid.", err)
		return
	}

	var env string
	if service.Env != nil {
		if ok, err := util.ValidateEnv(*service.Env); err != nil || !ok {
			writeError(w, http.StatusBadRequest, "Environment variables must be one KEY=value pair per line.", err)
			return
		}
		env = *service.Env
	}

	if service.Name == nil || *service.Name == "" {
		writeError(w, http.StatusBadRequest, "Service name is required.", nil)
		return
	}
	if service.Image == nil || *service.Image == "" {
		writeError(w, http.StatusBadRequest, "A Docker image is required.", nil)
		return
	}
	if service.Port == nil {
		writeError(w, http.StatusBadRequest, "A port is required.", nil)
		return
	}

	if service.Domain != nil {
		normalized := util.NormalizeDomain(*service.Domain)
		if normalized == "" {
			service.Domain = nil
		} else {
			if ok, _ := util.ValidateDomainLabel(normalized); !ok {
				writeError(w, http.StatusBadRequest, "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen.", nil)
				return
			}
			service.Domain = &normalized
		}
	}

	userId := claims.Subject

	domainRequested := service.Domain != nil

	res, err := app.Supabase.UpdateService(serviceId, userId, *service.Name, *service.Image, service.Domain, *service.Port)
	if err != nil {
		if domainRequested && isDomainTakenError(err) {
			writeError(w, http.StatusConflict, "That domain is already taken. Please choose a different one.", err)
			return
		}
		if domainRequested && isDomainValidationError(err) {
			writeError(w, http.StatusBadRequest, domainValidationMessage(err), err)
			return
		}
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "That name is already taken. Please choose a different one.", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Couldn't save the service.", err)
		return
	}

	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       util.ParseEnv(env),
		Image:     *service.Image,
		Port:      *service.Port,
	}); err != nil {
		if _, statusErr := app.Supabase.UpdateServiceStatus(res.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark service failed after deploy error", "service_id", res.Id, "err", statusErr)
		}
		writeError(w, http.StatusInternalServerError, "We saved your changes, but couldn't restart your service. A refresh will show its current status.", err)
		return
	}

	if _, err := app.Supabase.UpdateServiceStatus(res.Id, userId, "running"); err != nil {
		writeError(w, http.StatusInternalServerError, "Your service was restarted, but we couldn't confirm its status. A refresh will show where things stand.", err)
		return
	}
}

func (app *Application) DeleteService(w http.ResponseWriter, r *http.Request) {
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

	if err := app.Supabase.DeleteService(serviceId, claims.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't delete the service.", err)
		return
	}

	// The DB record is gone at this point regardless of what happens
	// below — log infra cleanup failures for ops rather than blocking
	// or confusing the user with a partial-failure response. (Previously
	// this branch returned with NO response written at all on a k8s
	// error, silently leaving the request hanging as an empty 200.)
	if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      "svc-" + serviceId,
	}); err != nil {
		slog.Error("failed to clean up service infrastructure", "service_id", serviceId, "err", err)
	}

	w.WriteHeader(204)
}

func (app *Application) DeleteServices(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That deletion request wasn't valid.", err)
		return
	}

	if len(req.Ids) == 0 {
		writeError(w, http.StatusBadRequest, "At least one service ID is required.", nil)
		return
	}

	deleted := []string{}
	failed := []FailedDelete{}
	for _, serviceId := range req.Ids {
		if err := app.Supabase.DeleteService(serviceId, claims.Subject); err != nil {
			failed = append(failed, FailedDelete{Id: serviceId, Error: "Couldn't delete the service."})
			continue
		}

		if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
			Namespace: "proj-" + projectId,
			Name:      "svc-" + serviceId,
		}); err != nil {
			slog.Error("failed to clean up service infrastructure", "service_id", serviceId, "err", err)
		}

		deleted = append(deleted, serviceId)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(BulkDeleteResponse{Deleted: deleted, Failed: failed}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
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
