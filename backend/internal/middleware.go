package internal

import "github.com/gin-gonic/gin"

func addMiddleware(g *gin.Engine) {
	g.Use(middlewareExample)
}

func middlewareExample(c *gin.Context) {
	// do something before request
	
	c.Next()
}
