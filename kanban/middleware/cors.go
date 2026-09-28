package middleware

import (
	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Max-InitData")
				c.Header("Access-Control-Expose-Headers", "Content-Length")
				c.Header("Vary", "Origin")
			}
			if c.Request.Method == "OPTIONS" {
				if _, ok := allowed[origin]; ok {
					c.AbortWithStatus(204)
				} else {
					c.AbortWithStatus(403)
				}
				return
			}
			if _, ok := allowed[origin]; !ok {
				c.AbortWithStatusJSON(403, gin.H{"error": "Origin not allowed"})
				return
			}
		} else if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
