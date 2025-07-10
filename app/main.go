package main

import (
	"fmt"
	"log"

	"github.com/chechetech/app/azure-go/db"
	Middlewares "github.com/chechetech/app/azure-go/middlewares"
	Routes "github.com/chechetech/app/azure-go/routes"

	"github.com/chechetech/app/azure-go/repositories/database"
	"github.com/gin-gonic/gin"
)

func main() {

	fmt.Println("starting...")

	// Connect to the database
	pool, err := db.Connect()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()
	fmt.Println("Successfully connected to the database!")

	clientset, err := Middlewares.InitializeClient()
	if err != nil {
		log.Fatalf("Failed to get client set: %v", err)
	}

	r := gin.Default()
	r.Use(Middlewares.SetClient(clientset))
	r.Use(Middlewares.ValidateToken())

	Routes.RegisterPodsRoutes(r)
	Routes.RegisterRegistriesRoutes(r)

	repos := database.NewRepositories(pool)
	Routes.RegisterDatabaseRoutes(r, repos)

	log.Println("Starting server v1")
	r.Run(":5000")
}
