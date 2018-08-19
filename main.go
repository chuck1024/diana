/**
 * Copyright 2018 Diana Author. All rights reserved.
 * Author: Chuck1024
 */

package main

import (
	"diana/service"
	"github.com/chuck1024/godog"
	_ "github.com/chuck1024/godog/log"
)

func main() {
	zkHost, _ := godog.AppConfig.String("zkHost")
	service.Service(zkHost)

	err := godog.Run()
	if err != nil {
		godog.Error("Error occurs, error = %s", err.Error())
		return
	}
}
