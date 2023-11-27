package server

import (
	"context"
	"fmt"
	ctrl "kms/tutorial/api/controller"
	srv "kms/tutorial/api/service"
	"kms/tutorial/common/config"
	"kms/tutorial/common/utils/errutil"
	_ "kms/tutorial/docs"
	"log"
	"math/big"
	"os"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

type Server struct {
	App *fiber.App
}

func newServer() *Server {
	var (
		curEnv = "dev"
	)

	app := fiber.New()
	app.Use(cors.New())
	app.Get("/swagger/*", swagger.HandlerDefault)

	rootPath := errutil.HandleFatal(os.Getwd())
	config.Init(rootPath + "/env/.env." + curEnv)
	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	cfg := errutil.HandleFatal(
		awscfg.LoadDefaultConfig(
			context.Background(),
			awscfg.WithCredentialsProvider(creds),
			awscfg.WithRegion(config.Env.AWS_REGION),
		),
	)
	chainID, ok := new(big.Int).SetString(config.Env.CHAIN_ID, 10)
	if !ok {
		log.Fatal("Invalid CHAIN_ID")
	}

	// service
	kmsSrv := srv.NewKmsSrv(cfg)
	txnSrv := srv.NewTxnSrv(chainID, kmsSrv)

	// controller
	apiRouter := app.Group("/api")
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
