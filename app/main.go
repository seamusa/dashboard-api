package main

import (
	"log"

	Middlewares "github.com/chechetech/app/azure-go/middlewares"
	Routes "github.com/chechetech/app/azure-go/routes"

	"github.com/gin-gonic/gin"
)

func main() {

	// fmt.Println(Middlewares.GenerateToken())

	clientset, err := Middlewares.InitializeClient()
	if err != nil {
		log.Fatalf("Failed to get client set: %v", err)
	}

	r := gin.Default()
	r.Use(Middlewares.SetClient(clientset))
	r.Use(Middlewares.ValidateToken())

	Routes.RegisterPodsRoutes(r)

	// Register RegistriesRoutes without ValidateToken middleware
	Routes.RegisterRegistriesRoutes(r)

	log.Println("Starting server v1")
	r.Run(":5000")
}
