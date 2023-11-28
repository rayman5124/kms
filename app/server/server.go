package server

import (
	"context"
	"fmt"
	ctrl "kms/wallet/app/api/controller"
	srv "kms/wallet/app/api/service"
	"kms/wallet/common/config"
	"kms/wallet/common/utils/errutil"
	_ "kms/wallet/docs"
	"log"
	"math/big"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

type Server struct {
	app *fiber.App
}

func newServer() *Server {
	app := fiber.New()
	app.Use(cors.New())
	app.Get("/swagger/*", swagger.HandlerDefault)

	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	awsCfg := errutil.HandleFatal(
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
	kmsSrv := srv.NewKmsSrv(awsCfg)
	txnSrv := srv.NewTxnSrv(chainID, kmsSrv)

	// controller
	apiRouter := app.Group("/api")
	ctrl.NewKmsCtrl(kmsSrv, apiRouter)
	ctrl.NewTxnCtrl(txnSrv, apiRouter)

	return &Server{app}
}

func Run(port string) chan error {
	server := newServer()
	ch := make(chan error)
	go func() { ch <- server.app.Listen(port) }()
	fmt.Printf("Listening at %v\n", port)
	return ch
}

// func Run(port string) {
// 	server := newServer()
// 	server.App.Listen(port)
// 	fmt.Printf("Listening at %v\n", port)
// }
