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
	"github.com/chuck1024/godog/dao/cache"
	"github.com/chuck1024/godog/utils"
)

const (
	sortSetPrefix = "diana:sortSet:"
	lockPrefix    = "diana:lock:"
	lockExpire    = 5
)

func getSortSetKey(num int) string {
	return fmt.Sprintf("%s%d", sortSetPrefix, num)
}

func getLockKey(num int) string {
	return fmt.Sprintf("%s%d", lockPrefix, num)
}

func SetLock(num int) (int, error) {
	key := getLockKey(num)
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

func ExpireLock(num int) (err error) {
	key := getLockKey(num)
	//godog.Debug("[ExpireLock] key: %s", key)

	if _, err := cache.Expire(key, lockExpire); err != nil {
		godog.Error("[ExpireLock] expire occur error: %s", err)
		return err
	}

	return
}

func DelLock(num int) (err error) {
	key := getLockKey(num)
	godog.Debug("[DelLock] key: %s", key)

	if _, err := cache.Del(key); err != nil {
		godog.Error("[DelLock] del occur error: %s", err)
		return err
	}

	return
}

func SetSortSet(num int, ts int64, value string) error {
	key := getSortSetKey(num)
	godog.Debug("[SetSortSet] key: %s", key)

	if err := cache.ZAdd(key, ts, value); err != nil {
		godog.Error("[SetSortSet] ZAdd occur error: %s", err)
		return err
	}

	return nil
}

func GetZCard(num int) (int, error) {
	key := getSortSetKey(num)
	//godog.Debug("[GetZCard] key: %s", key)

	length, err := cache.ZCard(key)
	if err != nil {
		godog.Error("[GetZCard] ZCard occur error: %s", err)
		return length, err
	}

	return length, nil
}

func GetZRange(num int) (string, error) {
	key := getSortSetKey(num)
	godog.Debug("[GetZRange] key: %s", key)

	value, err := cache.ZRange(key, 0, 0)
	if err != nil {
		godog.Error("[GetZRange] ZRange occur error: %s", err)
		return "", err
	}

	return value[0], nil
}

func DelSortSet(num int, value string) error {
	key := getSortSetKey(num)
	godog.Debug("[DelSortSet] key: %s", key)

	_, err := cache.ZRem(key, value)
	if err != nil {
		godog.Error("[DelSortSet] ZRange occur error: %s", err)
		return err
	}

	return nil
}
