package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zytan787/code-to-connect-2021/internal"
)

func main() {
	router := gin.Default()
	router.POST("/compress_trades", startCompressTrades)
	router.Run()
}

func startCompressTrades(c *gin.Context) {
	mainHandler := internal.NewMainHandler()
	mainHandler.CompressTrades(c)
}
