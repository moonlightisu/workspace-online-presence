package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type stubRealtime struct{}

func (stubRealtime) CreateChannel(context.Context, string, string) error { return nil }
func (stubRealtime) DeleteChannel(context.Context, string, string) error { return nil }
func (stubRealtime) IssueToken(context.Context, string, string, string) (json.RawMessage, error) {
	return json.RawMessage(`{"token":"client-token"}`), nil
}
func (stubRealtime) PublishAccountState(context.Context, string, string, string, string) error {
	return nil
}
func (stubRealtime) Presence(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(`{"members":[]}`), nil
}

func TestAuthorizeAccountLifecycle(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		arrange   func(*Service)
		tenantID  string
		accountID string
		wantErr   error
	}{
		{
			name: "active account in active tenant",
			arrange: func(s *Service) {
				mustOnboard(t, ctx, s, "acme")
				if _, err := s.RegisterAccount(ctx, "acme", "user-7", "register-7"); err != nil {
					t.Fatal(err)
				}
			},
			tenantID: "acme", accountID: "user-7",
		},
		{
			name: "disabled account is denied",
			arrange: func(s *Service) {
				mustOnboard(t, ctx, s, "acme")
				if _, err := s.RegisterAccount(ctx, "acme", "user-7", "register-7"); err != nil {
					t.Fatal(err)
				}
				if err := s.DisableAccount(ctx, "acme", "user-7", "disable-7"); err != nil {
					t.Fatal(err)
				}
			},
			tenantID: "acme", accountID: "user-7", wantErr: ErrAccountDisabled,
		},
		{
			name: "disabled tenant is denied",
			arrange: func(s *Service) {
				mustOnboard(t, ctx, s, "acme")
				if _, err := s.RegisterAccount(ctx, "acme", "user-7", "register-7"); err != nil {
					t.Fatal(err)
				}
				if err := s.DisableTenant(ctx, "acme", "disable-acme"); err != nil {
					t.Fatal(err)
				}
			},
			tenantID: "acme", accountID: "user-7", wantErr: ErrTenantDisabled,
		},
		{
			name: "unknown account is denied",
			arrange: func(s *Service) {
				mustOnboard(t, ctx, s, "acme")
			},
			tenantID: "acme", accountID: "missing", wantErr: ErrAccountMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(stubRealtime{})
			tt.arrange(s)
			if err := s.AuthorizeAccount(tt.tenantID, tt.accountID); !errors.Is(err, tt.wantErr) {
				t.Fatalf("AuthorizeAccount() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func mustOnboard(t *testing.T, ctx context.Context, s *Service, tenantID string) {
	t.Helper()
	if err := s.OnboardTenant(ctx, tenantID, "onboard-"+tenantID); err != nil {
		t.Fatal(err)
	}
}
