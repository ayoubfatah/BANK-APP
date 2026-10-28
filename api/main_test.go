package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	mockdb "github.com/bank-app/db/mock"
	"github.com/bank-app/db/store"
	"github.com/bank-app/token"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// for running tests

func RunCase(t *testing.T, tc TestCase, url string, method string) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mockdb.NewMockStore(ctrl)
	tc.buildStubs(st)

	server := newTestServer(t, st)

	// Create the request body if one is provided.
	var body io.Reader

	if tc.body != nil {
		data, err := json.Marshal(tc.body)
		require.NoError(t, err)

		body = bytes.NewReader(data)
	}

	// Create the HTTP request.
	request := httptest.NewRequest(method, url, body)
	request.Header.Set("Content-Type", "application/json")

	// Add authorization.
	tc.setupAuth(t, request, server.tokenMaker)

	// Execute the request.
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, request)

	// Check the response.
	tc.checkResponse(t, rec)
}

// Paseto symmetric keys must be exactly 32 bytes.
const testTokenKey = "12345678901234567890123456789012"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newTestServer(t *testing.T, st store.Store) *Server {
	config := utils.Config{
		TokenSymmetricKey:    testTokenKey,
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
		ServerAddress:        "127.0.0.1:0",
	}
	server, err := NewServer(config, st)
	require.NoError(t, err)
	return server
}

// ADAPT: match your token.Maker.CreateToken signature.
func addAuthorization(
	t *testing.T,
	request *http.Request,
	maker token.Maker,
	authType string,
	email string,
	userID int64,
	role string,
	duration time.Duration,
) {
	tok, _, err := maker.CreateToken(email, userID, role, duration)
	require.NoError(t, err)
	request.Header.Set(authorizationHeaderKey, fmt.Sprintf("%s %s", authType, tok))
}
