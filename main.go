/**
 * Copyright 2018 Diana Author. All rights reserved.
 * Author: Chuck1024
 */

package main

import (
	"diana/service"
	"godog"
	_ "godog/log"
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
