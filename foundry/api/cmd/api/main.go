package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/gin-gonic/gin"
)

var version = "dev"

type cli struct {
	Host string `short:"h" help:"Host to listen on" default:"0.0.0.0"`
	Port int    `short:"p" help:"Port to listen on" default:"8080"`
}

func main() {
	var c cli
	kong.Parse(&c)

	fmt.Printf("foundry version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello from Foundry!",
		})
	})

	listenAddr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	r.Run(listenAddr)
}
