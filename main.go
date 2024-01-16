package main

import (
	"flag"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/server"
	"kms/wallet/common/config"
	"kms/wallet/common/logger"
	"log"
	"os"

	"golang.org/x/exp/slices"
)

func main() {
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

	// if err := <-server.New.Run(":" + config.Env.PORT); err != nil {
	// 	fmt.Println(err)
	// }
	if err := server.New().Run(":7777"); err != nil {
		log.Fatal(err)
	}

}
