package main

import (
	"context"
	"flag"
	ctrl "kms/wallet/app/api/controller"
	"kms/wallet/app/api/model/dto"
	srv "kms/wallet/app/api/service"
	"kms/wallet/app/server"
	"kms/wallet/common/config"
	"kms/wallet/common/logger"
	"log"
	"math/big"
	"os"

	awscfg "github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go/aws"
	"golang.org/x/exp/slices"
)

func init() {
	var (
		curEnv = flag.String("env", "local", "environment")
	)
	flag.Parse()
	if envs := []string{"local", "dev", "stg", "prd"}; !slices.Contains(envs, *curEnv) {
		log.Fatalf("env must be one of %v (env=%v)", envs, *curEnv)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	logger.Init(*curEnv)
	config.Init(wd + "/env/.env." + *curEnv)
	dto.Init()
}

func main() {
	creds := credentials.NewStaticCredentialsProvider(config.Env.AWS_ACCESS_KEY, config.Env.AWS_SECRET_KEY, "")
	awsCfg, err := awscfg.LoadDefaultConfig(
		context.Background(),
		awscfg.WithCredentialsProvider(creds),
		awscfg.WithRegion(config.Env.AWS_REGION),
	)
	if err != nil {
		log.Fatal(err)
	}

	var kmsClient *kms.Client
	if config.Env.ENV == "local" {
		kmsClient = kms.NewFromConfig(awsCfg, func(o *kms.Options) {
			o.BaseEndpoint = aws.String("http://localhost:8080")
		})
	} else {
		kmsClient = kms.NewFromConfig(awsCfg)
	}

	server := server.New()
	chainID, ok := new(big.Int).SetString(config.Env.CHAIN_ID, 10)
	if !ok {
		log.Fatal("Invalid CHAIN_ID")
	}

	kmsSrv := srv.NewKmsSrv(kmsClient)
	txnSrv := srv.NewTxnSrv(chainID, kmsSrv)

	apiRouter := server.App.Group("/api")
	ctrl.NewAppCtrl().BootStrap(apiRouter)
	ctrl.NewKmsCtrl(kmsSrv).BootStrap(apiRouter)
	ctrl.NewTxnCtrl(txnSrv).BootStrap(apiRouter)

	if err := server.App.Listen(":7777"); err != nil {
		log.Fatal(err)
	}
}
