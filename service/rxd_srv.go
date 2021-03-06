/**
 * Copyright 2018 Author. All rights reserved.
 * Author: Chuck1024
 */

package service

import (
	"encoding/json"
	"github.com/chuck1024/diana/dao/cache"
	"github.com/chuck1024/diana/model"
	"github.com/chuck1024/godog"
)

func Rxd(req *model.RxdReq, currentTs int64) error {
	reqByte, err := json.Marshal(req)
	if err != nil {
		godog.Error("[Rxd] json marshal occur error: %s", err)
		return err
	}

	SortSetNum, _ := godog.AppConfig.Int("sortSetNum")

	num := int(req.Uuid) % SortSetNum

	if err = cache.SetSortSet(num, currentTs, string(reqByte)); err != nil {
		godog.Error("[Rxd] cache.SetSortSet occur error: %s", err)
		return err
	}

	return nil
}
