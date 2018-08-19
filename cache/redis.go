/**
 * Copyright 2018 Diana Author. All rights reserved.
 * Author: Chuck1024
 */

package cache

import (
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/chuck1024/godog"
	"github.com/chuck1024/godog/store/cache"
	"github.com/chuck1024/godog/utils"
)

const (
	listPrefix = "diana:list:"
	lockPrefix = "diana:lock:"
	lockExpire = 5
)

func getListKey(list int) string {
	return fmt.Sprintf("%s%d", listPrefix, list)
}

func getLockKey(list int) string {
	return fmt.Sprintf("%s%d", lockPrefix, list)
}

func SetLock(list int) (int, error) {
	key := getLockKey(list)
	//godog.Debug("[SetLock] key: %s", key)

	value := utils.GetLocalIP()
	num, err := cache.SetNx(key, value)
	if err != nil {
		//godog.Error("[SetLock] setNx occur error: %s", err)
		return 0, err
	}

	if num == 0 {
		return 0, errors.New("setNx occur error")
	}

	if _, err := cache.Expire(key, lockExpire); err != nil {
		beego.Error("[SetLock] expire occur error: ", err)
		return 0, err
	}

	return num, nil
}

func ExpireLock(list int) (err error) {
	key := getLockKey(list)
	//godog.Debug("[ExpireLock] key: %s", key)

	if _, err := cache.Expire(key, lockExpire); err != nil {
		godog.Error("[ExpireLock] expire occur error: %s", err)
		return err
	}

	return
}

func DelLock(list int) (err error) {
	key := getLockKey(list)
	godog.Debug("[DelLock] key: %s", key)

	if _, err := cache.Del(key); err != nil {
		godog.Error("[DelLock] del occur error: %s", err)
		return err
	}

	return
}

func GetListLen(list int) (int, error) {
	key := getListKey(list)
	//godog.Debug("[GetListLen] key: %s", key)

	length, err := cache.LLen(key)
	if err != nil {
		godog.Error("[GetListLen] LLen occur error:%s ", err)
		return 0, err
	}

	return length, nil
}

func GetListRPop(list int) (string, error) {
	key := getListKey(list)
	//godog.Debug("[GetListPop] key: %s", key)

	value, err := cache.RPop(key)
	if err != nil {
		godog.Error("[GetListPop] RPop occur error: %s", err)
		return "", err
	}

	return string(value), nil
}

func SetListLPush(list int, value string) error {
	key := getListKey(list)
	godog.Debug("[SetListLPush] key: %s", key)

	if _, err := cache.LPush(key, value); err != nil {
		godog.Error("[SetListLPush] LPush occur error: %s", err)
		return err
	}

	return nil
}
