package main

import (
	"fmt"
	"kms/tutorial/api/server"
)

type AwsInfo struct {
	Accesskey string
	Secretkey string
}

var keyid string = "f50a9229-e7c7-45ba-b06c-8036b894424e"

func main() {
	if err := <-server.Run(":7777"); err != nil {
		fmt.Println(err)
	}
	// server.Run(":7777")

}
