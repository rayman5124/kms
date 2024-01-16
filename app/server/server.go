package server

import (
	"context"
	"encoding/json"
	"fmt"
	ctrl "kms/wallet/app/api/controller"
	srv "kms/wallet/app/api/service"
	"kms/wallet/common/config"
	"kms/wallet/common/logger"
	"kms/wallet/common/utils/timeutil"
	_ "kms/wallet/docs"
	"math/big"
	"time"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
)

type server struct {
	App *fiber.App
}

func New() *server {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrHandler,
	})
	app.Use(cors.New())
	app.Use(limiter.New(limiter.Config{
		Expiration: 60 * time.Second,
		Max:        1000,
	}))
	// handle panic
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			parser := func(data []byte) (ret any) {
				json.Unmarshal(data, &ret)
				return
			}
			data := map[string]any{
				"ip":          c.IP(),
				"path":        c.Path(),
				"method":      c.Method(),
				"reqHeader":   c.GetReqHeaders(),
				"queryParams": parser(c.Context().URI().QueryString()),
				"reqBody":     string(c.Request().Body()),
			}
			// zlogger.Panic().Err(fmt.Errorf("%v", e)).Interface("data", data).Send()
			logger.Panic().E(fmt.Errorf("%v", e)).D("data", data).W()
			// fmt.Printf("[Panic] \r\nip: %s \r\nstatus: %v \r\npath: %s \r\nmethod: %s \r\nreqHeader: %s \r\nqueryParams: %s \r\nreqBody: %s \r\nresBody: %s \r\ntime: %s \r\nerrLog: %s\r\n\r\n", c.IP(), c.Response().StatusCode(), c.Path(), c.Method(), strings.ReplaceAll(strings.ReplaceAll(string(c.Request().Header.RawHeaders()), "\r\n", "&"), ": ", "="), c.Request().URI().QueryString(), c.Request().Body(), c.Response().Body(), timeutil.FormatNow(), fmt.Sprintf("%v\n%s\n", e, debug.Stack()))

		},
	}))
	// swagger
	app.Get("/swagger/*", swagger.HandlerDefault) // logger 세팅 전에 설정

	// logger
	if config.Env.Log {
		app.Use(fiberlogger.New(fiberlogger.Config{ // Only all routes that are registered after this one will be logged
			Format:     formatter(),
			TimeFormat: timeutil.DateFormat,
			Output:     &writer{},
		}))
	}
	// kms client
	var kmsClient *kms.Client
	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	awsCfg, err := awscfg.LoadDefaultConfig(
		context.Background(),
		awscfg.WithCredentialsProvider(creds),
		awscfg.WithRegion(config.Env.AWS_REGION),
	)
	if err != nil {
		log.Fatal(err)
	}

	if config.Env.ENV == "local" {
		kmsClient = kms.NewFromConfig(awsCfg, func(o *kms.Options) {
			o.BaseEndpoint = aws.String("http://localhost:8080")
		})
	} else {
		kmsClient = kms.NewFromConfig(awsCfg)
	}

	chainID, ok := new(big.Int).SetString(config.Env.CHAIN_ID, 10)
	if !ok {
		log.Fatal("Invalid CHAIN_ID")
	}

	// service
	kmsSrv := srv.NewKmsSrv(kmsClient)
	txnSrv := srv.NewTxnSrv(chainID, kmsSrv)

	// controller
	apiRouter := app.Group("/api")
	ctrl.NewAppCtrl().BootStrap(apiRouter)
	ctrl.NewKmsCtrl(kmsSrv).BootStrap(apiRouter)
	ctrl.NewTxnCtrl(txnSrv).BootStrap(apiRouter)

	return &server{app}
}

// func (s *server) Run(port string) <-chan error {
// 	ch := make(chan error)
// 	go func() { ch <- s.App.Listen(port) }()
// 	fmt.Printf("Listening at %v\n", port)
// 	return ch
// }

func (s *server) Run(port string) error {
	return s.App.Listen(port)
}
