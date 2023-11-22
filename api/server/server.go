package server

import (
	"fmt"
	ctrl "kms/tutorial/api/controller"
	srv "kms/tutorial/api/service"
	"kms/tutorial/common/config"
	_ "kms/tutorial/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

type Server struct {
	App *fiber.App
}

func newServer() *Server {
	config.Init("dev")
	app := fiber.New()
	app.Use(cors.New())
	app.Get("/swagger/*", swagger.HandlerDefault)

	// service
	kmsSrv := srv.NewKmsSrv(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, config.Env.AWS_REGION)
	txnSrv := srv.NewTxnSrv(config.Env.RPC_END_POINT, kmsSrv)

	apiRouter := app.Group("/api")

	// controller
	ctrl.NewKmsCtrl(kmsSrv, apiRouter)
	ctrl.NewTxnCtrl(txnSrv, apiRouter)

	return &Server{App: app}
}

func Run(port string) chan error {
	server := newServer()
	ch := make(chan error)
	go func() { ch <- server.App.Listen(port) }()
	fmt.Printf("Listening at %v\n", port)
	return ch
}

// func Run(port string) {
// 	server := newServer()
// 	server.App.Listen(port)
// 	fmt.Printf("Listening at %v\n", port)
// }
