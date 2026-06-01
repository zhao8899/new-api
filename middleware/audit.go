package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const AuditDiffContextKey = "audit_diff"

func AuditAction(action string, resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.GetInt("id") <= 0 {
			return
		}
		statusCode := c.Writer.Status()
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		result := "success"
		if statusCode >= http.StatusBadRequest {
			result = "failure"
		}
		resourceID := strings.TrimSpace(c.Param("id"))
		if resourceID == "" {
			resourceID = strings.TrimSpace(c.GetString("resource_id"))
		}
		if _, err := model.RecordAuditEvent(model.AuditEventParams{
			ActorID:      c.GetInt("id"),
			ActorRole:    c.GetInt("role"),
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			SourceIP:     c.ClientIP(),
			RequestID:    c.GetString(common.RequestIdKey),
			Result:       result,
			DiffRedacted: c.GetString(AuditDiffContextKey),
			Method:       c.Request.Method,
			Path:         c.FullPath(),
			StatusCode:   statusCode,
		}); err != nil {
			common.SysLog(fmt.Sprintf("failed to record audit event: %s", common.RedactSensitiveText(err.Error())))
		}
	}
}
