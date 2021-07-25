package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/zytan787/code-to-connect-2021/internal"
	"os"
	"time"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("FRONTEND_HOST")},
		AllowMethods:     []string{"POST"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.POST("/compress_trades", startCompressTrades)
	router.Run()
}

func startCompressTrades(c *gin.Context) {
	mainHandler := internal.NewMainHandler()
	mainHandler.CompressTrades(c)
}
