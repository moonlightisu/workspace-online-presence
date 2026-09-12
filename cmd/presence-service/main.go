package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"example.com/workspace-presence/internal/workspace"
)

type server struct {
	service *workspace.Service
}

type command struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	RequestID string `json:"request_id"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	s := &server{service: workspace.NewService(workspace.NewClient(key))}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/tenants", s.onboardTenant)
	mux.HandleFunc("DELETE /admin/tenants/{tenantID}", s.disableTenant)
	mux.HandleFunc("POST /admin/tenants/{tenantID}/accounts", s.registerAccount)
	mux.HandleFunc("DELETE /admin/tenants/{tenantID}/accounts/{accountID}", s.disableAccount)
	mux.HandleFunc("GET /workspaces/{tenantID}/online", s.online)

	addr := ":8080"
	log.Printf("workspace presence listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) onboardTenant(w http.ResponseWriter, r *http.Request) {
	var cmd command
	if !decode(w, r, &cmd) {
		return
	}
	if err := s.service.OnboardTenant(r.Context(), cmd.TenantID, cmd.RequestID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"tenant_id": cmd.TenantID, "state": "active"})
}

func (s *server) registerAccount(w http.ResponseWriter, r *http.Request) {
	var cmd command
	if !decode(w, r, &cmd) {
		return
	}
	data, err := s.service.RegisterAccount(r.Context(), r.PathValue("tenantID"), cmd.AccountID, cmd.RequestID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeRaw(w, http.StatusCreated, data)
}

func (s *server) disableAccount(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("Idempotency-Key")
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Idempotency-Key is required"})
		return
	}
	err := s.service.DisableAccount(r.Context(), r.PathValue("tenantID"), r.PathValue("accountID"), requestID)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) disableTenant(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("Idempotency-Key")
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Idempotency-Key is required"})
		return
	}
	if err := s.service.DisableTenant(r.Context(), r.PathValue("tenantID"), requestID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) online(w http.ResponseWriter, r *http.Request) {
	data, err := s.service.Online(r.Context(), r.PathValue("tenantID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeRaw(w, http.StatusOK, data)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	switch {
	case errors.Is(err, workspace.ErrTenantMissing), errors.Is(err, workspace.ErrAccountMissing):
		status = http.StatusNotFound
	case errors.Is(err, workspace.ErrTenantDisabled), errors.Is(err, workspace.ErrAccountDisabled):
		status = http.StatusConflict
	default:
		var apiErr *workspace.InfraiError
		if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
			status = apiErr.Status
		} else if strings.Contains(err.Error(), "required") {
			status = http.StatusBadRequest
		}
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeRaw(w http.ResponseWriter, status int, data json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
