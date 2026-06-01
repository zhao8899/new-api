package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecureVerificationRequiredRejectsMissingVerification(t *testing.T) {
	router := newSecureVerificationTestRouter(t, nil)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/critical", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "VERIFICATION_REQUIRED")
}

func TestSecureVerificationRequiredAllowsFreshVerification(t *testing.T) {
	router := newSecureVerificationTestRouter(t, func(session sessions.Session) {
		session.Set(SecureVerificationSessionKey, time.Now().Unix())
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/critical", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "ok")
}

func TestSecureVerificationRequiredRejectsExpiredVerification(t *testing.T) {
	router := newSecureVerificationTestRouter(t, func(session sessions.Session) {
		session.Set(SecureVerificationSessionKey, time.Now().Unix()-SecureVerificationTimeout-1)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/critical", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "VERIFICATION_EXPIRED")
}

func newSecureVerificationTestRouter(t *testing.T, seed func(sessions.Session)) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("secure-verification-test"))))
	router.Use(func(c *gin.Context) {
		c.Set("id", 1)
		if seed != nil {
			session := sessions.Default(c)
			seed(session)
			require.NoError(t, session.Save())
		}
		c.Next()
	})
	router.POST("/critical", SecureVerificationRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "ok"})
	})
	return router
}
