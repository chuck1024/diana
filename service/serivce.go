/**
 * Copyright 2018 Author. All rights reserved.
 * Author: Chuck1024
 */

package service

import (
	"godog"
)

func Service(zkHost string) {
	if err := connectZk(zkHost); err != nil {
		godog.Error("[start] connectZK occur error:%s", err)
		return
	}

	go watch()
	go manager()
}
