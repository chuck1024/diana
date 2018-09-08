/**
 * Copyright 2018 Diana Author. All rights reserved.
 * Author: Chuck1024
 */

package main

import (
	"diana/model/service"
	"github.com/chuck1024/godog"
	"github.com/chuck1024/godog/dao/cache"
	_ "github.com/chuck1024/godog/log"
)

func main() {
	url, _ := godog.AppConfig.String("redis")
	cache.Init(url)

	zkHost, _ := godog.AppConfig.String("zkHost")
	service.Service(zkHost)

	err := godog.Run()
	if err != nil {
		godog.Error("Error occurs, error = %s", err.Error())
		return
	}
}
