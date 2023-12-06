package server

import (
	"context"
	ctrl "kms/wallet/app/api/controller"
	srv "kms/wallet/app/api/service"
	"kms/wallet/common/config"
	"kms/wallet/common/utils/errutil"
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
	"github.com/gofiber/fiber/v2/middleware/logger"
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
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	// swagger
	app.Get("/swagger/*", swagger.HandlerDefault) // logger 세팅 전에 설정
	// logger
	if config.Env.Log {
		app.Use(logger.New(logger.Config{ // Only all routes that are registered after this one will be logged
			CustomTags: map[string]logger.LogFunc{
				"level": func(output logger.Buffer, c *fiber.Ctx, data *logger.Data, extraParam string) (int, error) {
					level := "[Info]"
					if levelFromErrHandler := c.Locals("level"); levelFromErrHandler != nil {
						level = levelFromErrHandler.(string)
						if level == "[Info]" {
							data.ChainErr = nil // Info레벨은 에러로그를 찍지 않기 위함
						}
					}
					return output.WriteString(level)
				},
			},
			Format:     "${level} \r\nip: ${ip} \r\nstatus: ${status} \r\npath: ${path} \r\nmethod: ${method} \r\nreqHeader: ${reqHeader} \r\nqueryParams: ${queryParams} \r\nreqBody: ${body} \r\nresBody: ${resBody} \r\ntime: ${time} \r\nlatency: ${latency} \r\nerrLog: ${error}\r\n\r\n",
			TimeFormat: timeutil.DateFormat,
		}))
	}
	// kms client
	var kmsClient *kms.Client
	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	awsCfg :=
		errutil.HandleFatal(awscfg.LoadDefaultConfig(
			context.Background(),
			awscfg.WithCredentialsProvider(creds),
			awscfg.WithRegion(config.Env.AWS_REGION),
		))
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
