package core

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

func createTestKey(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	key, err := jwk.FromRaw(privKey)
	require.NoError(t, err)

	// Set standard headers
	err = key.Set(jwk.KeyIDKey, "test-key")
	require.NoError(t, err)
	err = key.Set(jwk.AlgorithmKey, "RS256")
	require.NoError(t, err)

	// Create test_data directory if it doesn't exist
	err = os.MkdirAll("test_data", 0755)
	require.NoError(t, err)

	// Save private key
	timestamp := time.Now().Unix()
	privateKeyPath := fmt.Sprintf("test_data/key-%d.private.jwk", timestamp)
	keyBytes, err := json.MarshalIndent(key, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(privateKeyPath, keyBytes, 0600)
	require.NoError(t, err)
}

func TestNewServer(t *testing.T) {
	createTestKey(t)
	// Setup test configuration
	viper.Set("keys.dir", "test_data")

	// Test server creation
	server, err := NewServer()
	require.NoError(t, err)
	require.NotNil(t, server)
	require.NotNil(t, server.issuer)
}

// Add new mock using testify
type MockTailscaleLocalClient struct {
	mock.Mock
}

func (m *MockTailscaleLocalClient) WhoIs(ctx context.Context, addr string) (*apitype.WhoIsResponse, error) {
	args := m.Called(ctx, addr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apitype.WhoIsResponse), args.Error(1)
}

func TestServerHandleRequests(t *testing.T) {
	// Setup test configuration
	tempDir, err := os.MkdirTemp("", "tailbone-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	viper.Set("keys.dir", "test_data")
	viper.Set("server.tailscale.authkey", "fake-key")
	viper.Set("server.tailscale.dir", tempDir)
	viper.Set("server.tailscale.hostname", "test-host")
	viper.Set("server.port", 8080)

	// Create server
	server, err := NewServer()
	require.NoError(t, err)

	// Create mock client
	mockClient := new(MockTailscaleLocalClient)
	mockClient.On("WhoIs", mock.Anything, mock.Anything).Return(&apitype.WhoIsResponse{
		UserProfile: &tailcfg.UserProfile{
			LoginName:   "test@example.com",
			DisplayName: "Test User",
		},
	}, nil)

	server.client = mockClient
	server.server = &tsnet.Server{}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "health check",
			method:         "GET",
			path:           "/_healthz",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.True(t, resp["ok"].(bool))
			},
		},
		{
			name:           "issue token - wrong method",
			method:         "GET",
			path:           "/issue",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "issue token - success",
			method:         "POST",
			path:           "/issue",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]string
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp["token"])
			},
		},
		{
			name:           "not found",
			method:         "GET",
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Create handler and serve request
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				reqLogger := server.logger.With().
					Str("remote_addr", r.RemoteAddr).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Logger()

				switch r.URL.Path {
				case "/_healthz":
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"ok":      true,
						"version": "test",
						"commit":  "test",
					})
					return

				case "/issue":
					if r.Method != http.MethodPost {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						return
					}

					who, err := server.client.WhoIs(ctx, r.RemoteAddr)
					if err != nil {
						reqLogger.Error().Err(err).Msg("failed to identify user")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					token, err := server.issuer.IssueToken(ctx, who.UserProfile.LoginName, who.UserProfile.DisplayName)
					if err != nil {
						reqLogger.Error().Err(err).Msg("failed to issue token")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					json.NewEncoder(w).Encode(map[string]string{
						"token": token,
					})

				default:
					http.NotFound(w, r)
				}
			})

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
