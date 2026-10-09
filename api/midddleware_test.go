package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bank-app/token"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testSymmetricKey = "12345678901234567890123456789012"

func newTestMaker(t *testing.T) *token.PasetoMaker {
	t.Helper()

	maker, err := token.NewPasetoMaker(testSymmetricKey)
	require.NoError(t, err)

	pasetoMaker, ok := maker.(*token.PasetoMaker)
	require.True(t, ok)

	return pasetoMaker
}

func createTestToken(
	t *testing.T,
	maker *token.PasetoMaker,
	role string,
) string {
	t.Helper()

	tokenString, err := maker.CreateToken(
		"test@example.com",
		1,
		role,
		time.Minute,
	)
	require.NoError(t, err)

	return tokenString
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	maker := newTestMaker(t)
	validToken := createTestToken(t, maker, "customer")
	payload, err := maker.VerifyToken(validToken)
	require.NoError(t, err)

	t.Logf("Decoded payload: %+v", payload)
	require.Equal(t, "customer", payload.Role)


	testCases := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid customer token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "No header was passed",
		},
		{
			name:           "Missing token",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid or Missing Bearer Token",
		},
		{
			name:           "Extra header fields",
			authHeader:     "Bearer " + validToken + " extra",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid or Missing Bearer Token",
		},
		{
			name:           "Unsupported authorization type",
			authHeader:     "Basic " + validToken,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authorization Type Not Supported",
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Access Token Not Valid",
		},
		{
			name:           "Lowercase bearer",
			authHeader:     "bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPayload *token.Payload
			handlerCalled := false

			router := gin.New()

			router.GET(
				"/protected",
				authMiddleware(maker),
				func(ctx *gin.Context) {
					handlerCalled = true
					gotPayload = getAuthPayload(ctx)
					ctx.Status(http.StatusOK)
				},
			)

			req := httptest.NewRequest(
				http.MethodGet,
				"/protected",
				nil,
			)

			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			require.Equal(t, tc.expectedStatus, recorder.Code)

			if tc.expectedError != "" {
				require.Contains(
					t,
					recorder.Body.String(),
					tc.expectedError,
				)
			}

			if tc.expectedStatus == http.StatusOK {
				require.True(t, handlerCalled)
				require.NotNil(t, gotPayload)
				require.Equal(t, "customer", gotPayload.Role)
				require.Equal(t, int64(1), gotPayload.UserID)
				require.Equal(t, "test@example.com", gotPayload.Email)
			} else {
				require.False(t, handlerCalled)
			}
		})
	}
}

func TestAdminMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	maker := newTestMaker(t)

	testCases := []struct {
		name           string
		role           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Admin access granted",
			role:           "admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Customer access denied",
			role:           "customer",
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
		{
			name:           "Uppercase admin access granted",
			role:           "ADMIN",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authToken := createTestToken(t, maker, tc.role)
			handlerCalled := false

			router := gin.New()

			router.GET(
				"/admin",
				authMiddleware(maker),
				adminMiddleware(),
				func(ctx *gin.Context) {
					handlerCalled = true
					ctx.Status(http.StatusOK)
				},
			)

			req := httptest.NewRequest(
				http.MethodGet,
				"/admin",
				nil,
			)
			req.Header.Set("Authorization", "Bearer "+authToken)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			require.Equal(t, tc.expectedStatus, recorder.Code)

			if tc.expectedError != "" {
				require.Contains(
					t,
					recorder.Body.String(),
					tc.expectedError,
				)
			}

			if tc.expectedStatus == http.StatusOK {
				require.True(t, handlerCalled)
			} else {
				require.False(t, handlerCalled)
			}
		})
	}
}
