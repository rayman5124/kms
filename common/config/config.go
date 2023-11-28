package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type EnvStruct struct {
	ENV            string
	PORT           string
	CHAIN_ID       string
	AWS_ACCESS_KEY string
	AWS_SECRET_KEY string
	AWS_REGION     string
}

var Env *EnvStruct

func Init(envPath string) {
	if Env == nil {
		Env = new(EnvStruct)
	}

	if err := godotenv.Load(envPath); err != nil {
		fmt.Println(err)
		log.Fatalf("Failed to load %s env file\n", envPath)
	}
	Env.ENV = getEnv("ENV", true)
	Env.PORT = getEnv("PORT", true)
	Env.CHAIN_ID = getEnv("CHAIN_ID", true)
	Env.AWS_ACCESS_KEY = getEnv("AWS_ACCESS_KEY", true)
	Env.AWS_SECRET_KEY = getEnv("AWS_SECRET_KEY", true)
	Env.AWS_REGION = getEnv("AWS_REGION", true)

	envJson, _ := json.MarshalIndent(Env, "", "\t")
	fmt.Println(string(envJson))
}

func getEnv(key string, required bool) string {
	val, success := os.LookupEnv(key)
	if required && !success {
		log.Fatalf("Required value of %s dosen't exist", key)
	}

	return val
}
