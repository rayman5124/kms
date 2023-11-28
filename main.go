package main

import (
	"flag"
	"fmt"
	"kms/wallet/app/server"
	"kms/wallet/common/config"
	"kms/wallet/common/utils/errutil"
	"log"
	"os"

	"golang.org/x/exp/slices"
)

func main() {
	var (
		curEnv = flag.String("env", "dev", "environment")
	)

	flag.Parse()
	if envs := []string{"local", "dev", "stg", "prd"}; !slices.Contains(envs, *curEnv) {
		log.Fatalf("env must be one of %v (env=%v)", envs, *curEnv)
	}
	rootPath := errutil.HandleFatal(os.Getwd())
	config.Init(rootPath + "/env/.env." + *curEnv)

	if err := <-server.Run(":" + config.Env.PORT); err != nil {
		fmt.Println(err)
	}
	// server.Run(":7777")
}
