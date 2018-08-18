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

var App *godog.Application

func main() {
	App = godog.NewApplication("diana")
	zkHost, _ := App.AppConfig.String("zkHost")
	service.Service(zkHost)

	err := App.Run()
	if err != nil {
		godog.Error("Error occurs, error = %s", err.Error())
		return
	}
}
