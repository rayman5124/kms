package main

import (
	"flag"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/server"
	"kms/wallet/common/config"
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

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Init(wd + "/env/.env." + *curEnv); err != nil {
		log.Fatal(err)
	}
	dto.Init()

	// if err := <-server.Run(":" + config.Env.PORT); err != nil {
	// 	fmt.Println(err)
	// }
	if err := server.Run(":7777"); err != nil {
		log.Fatal(err)
	}
}
