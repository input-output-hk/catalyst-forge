package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// KratosLoginFlow proxies the Kratos login flow JSON without forwarding cookies.
// It is intended for SPAs to read the flow (including csrf_token and ui.action)
// without tripping CSRF protections.
func (h *Handlers) KratosLoginFlow(c *gin.Context) {
	flowID := c.Query("id")
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id", "correlation": corrFields(c)})
		return
	}

	endpoint := fmt.Sprintf("%s/self-service/login/flows?id=%s", h.cfg.Kratos.PublicURL, url.QueryEscape(flowID))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}
	// Ensure no cookies are forwarded; set minimal headers
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	// Disable caching of flow JSON
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
	c.Data(resp.StatusCode, contentType, body)
}
