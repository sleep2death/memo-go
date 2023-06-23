package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sleep2death/memo-go"
	_ "github.com/sleep2death/memo-go/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// gin-swagger middleware
// swagger embed files

//	@title			Swagger Memo API
//	@version		0.0.1
//	@description	This is a server for memo-services.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://github.com/sleep2death/memo-go/issues
//	@contact.email	aspirin2d@outlook.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.basic	BasicAuth

//	@externalDocs.description	OpenAPI
//	@externalDocs.url			https://swagger.io/resources/open-api/
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	v1 := r.Group("/api/v1")

	handlers, err := memo.New()
	if err != nil {
		log.Fatal(err)
	}

	v1.GET("/s", handlers.GetSessions)
	v1.POST("/s/add", handlers.AddSession)
	v1.DELETE("/s/:id/del", handlers.DeleteSession)
	v1.GET("/s/:id", handlers.GetSession)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run()
}
