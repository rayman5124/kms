package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type EnvStruct struct {
	RPC_END_POINT  string
	AWS_ACCESS_KEY string
	AWS_SECRET_KEY string
	AWS_REGION     string
}

var Env *EnvStruct

func Init(curEnv string) {
	if Env == nil {
		Env = new(EnvStruct)
	}

	if err := godotenv.Load("./env/.env." + curEnv); err != nil {
		log.Fatalf("Failed to load %s env file\n", curEnv)
	}

	Env.RPC_END_POINT = getEnv("RPC_END_POINT", true)
	Env.AWS_ACCESS_KEY = getEnv("AWS_ACCESS_KEY", true)
	Env.AWS_SECRET_KEY = getEnv("AWS_SECRET_KEY", true)
	Env.AWS_REGION = getEnv("AWS_REGION", true)
}

func getEnv(key string, required bool) string {
	val, success := os.LookupEnv(key)
	if required && !success {
		log.Fatalf("Required value of %s dosen't exist", key)
	}

	return val
}
