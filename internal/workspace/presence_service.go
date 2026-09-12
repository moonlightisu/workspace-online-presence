package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrTenantMissing   = errors.New("tenant is not onboarded")
	ErrTenantDisabled  = errors.New("tenant is disabled")
	ErrAccountMissing  = errors.New("account is not registered")
	ErrAccountDisabled = errors.New("account is disabled")
)

type RealtimeAPI interface {
	CreateChannel(context.Context, string, string) error
	DeleteChannel(context.Context, string, string) error
	IssueToken(context.Context, string, string, string) (json.RawMessage, error)
	PublishAccountState(context.Context, string, string, string, string) error
	Presence(context.Context, string) (json.RawMessage, error)
}

type tenant struct {
	active   bool
	accounts map[string]bool
}

type Service struct {
	mu      sync.RWMutex
	tenants map[string]*tenant
	api     RealtimeAPI
}

func NewService(api RealtimeAPI) *Service {
	return &Service{api: api, tenants: make(map[string]*tenant)}
}

func (s *Service) OnboardTenant(ctx context.Context, tenantID, requestID string) error {
	if tenantID == "" || requestID == "" {
		return errors.New("tenant_id and request_id are required")
	}
	if err := s.api.CreateChannel(ctx, channelFor(tenantID), requestID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.tenants[tenantID]; existing != nil {
		existing.active = true
		return nil
	}
	s.tenants[tenantID] = &tenant{active: true, accounts: make(map[string]bool)}
	return nil
}

func (s *Service) RegisterAccount(ctx context.Context, tenantID, accountID, requestID string) (json.RawMessage, error) {
	if accountID == "" || requestID == "" {
		return nil, errors.New("account_id and request_id are required")
	}
	s.mu.Lock()
	t, err := s.usableTenantLocked(tenantID)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	t.accounts[accountID] = true
	s.mu.Unlock()

	token, err := s.api.IssueToken(ctx, accountID, channelFor(tenantID), requestID)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *Service) DisableAccount(ctx context.Context, tenantID, accountID, requestID string) error {
	s.mu.Lock()
	t, err := s.usableTenantLocked(tenantID)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if _, ok := t.accounts[accountID]; !ok {
		s.mu.Unlock()
		return ErrAccountMissing
	}
	t.accounts[accountID] = false
	s.mu.Unlock()
	return s.api.PublishAccountState(ctx, channelFor(tenantID), accountID, "disabled", requestID)
}

func (s *Service) DisableTenant(ctx context.Context, tenantID, requestID string) error {
	if requestID == "" {
		return errors.New("request_id is required")
	}
	s.mu.RLock()
	_, ok := s.tenants[tenantID]
	s.mu.RUnlock()
	if !ok {
		return ErrTenantMissing
	}
	if err := s.api.DeleteChannel(ctx, channelFor(tenantID), requestID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[tenantID].active = false
	return nil
}

func (s *Service) Online(ctx context.Context, tenantID string) (json.RawMessage, error) {
	s.mu.RLock()
	t, ok := s.tenants[tenantID]
	active := ok && t.active
	s.mu.RUnlock()
	if !ok {
		return nil, ErrTenantMissing
	}
	if !active {
		return nil, ErrTenantDisabled
	}
	return s.api.Presence(ctx, channelFor(tenantID))
}

func (s *Service) AuthorizeAccount(tenantID, accountID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[tenantID]
	if !ok {
		return ErrTenantMissing
	}
	if !t.active {
		return ErrTenantDisabled
	}
	active, ok := t.accounts[accountID]
	if !ok {
		return ErrAccountMissing
	}
	if !active {
		return ErrAccountDisabled
	}
	return nil
}

func (s *Service) usableTenantLocked(tenantID string) (*tenant, error) {
	t, ok := s.tenants[tenantID]
	if !ok {
		return nil, ErrTenantMissing
	}
	if !t.active {
		return nil, ErrTenantDisabled
	}
	return t, nil
}

func channelFor(tenantID string) string {
	return fmt.Sprintf("workspace:%s", tenantID)
}
