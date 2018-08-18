/**
 * Copyright 2018 Diana Author. All rights reserved.
 * Author: Chuck1024
 */

package cache

import (
	"fmt"
	"godog"
	"godog/store/cache"
	"godog/utils"
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

func SetLock(list int) error {
	key := getLockKey(list)
	//godog.Debug("[SetLock] key: %s", key)

	value := utils.GetLocalIP()
	err := cache.RedisHandle.Set(key, value, lockExpire, 0, false, true)
	if err != nil {
		//godog.Error("[SetLock] setNx occur error: %s", err)
		return err
	}

	return nil
}

func ExpireLock(list int) (err error) {
	key := getLockKey(list)
	//godog.Debug("[ExpireLock] key: %s", key)

	if _, err := cache.RedisHandle.ExecuteCommand("EXPIRE", key, lockExpire); err != nil {
		godog.Error("[ExpireLock] expire occur error: %s", err)
		return err
	}

	return
}

func DelLock(list int) (err error) {
	key := getLockKey(list)
	godog.Debug("[DelLock] key: %s", key)

	if _, err := cache.RedisHandle.Del(key); err != nil {
		godog.Error("[DelLock] del occur error: %s", err)
		return err
	}

	return
}

func GetListLen(list int) (int64, error) {
	key := getListKey(list)
	//godog.Debug("[GetListLen] key: %s", key)

	length, err := cache.RedisHandle.LLen(key)
	if err != nil {
		godog.Error("[GetListLen] LLen occur error:%s ", err)
		return 0, err
	}

	return length, nil
}

func GetListRPop(list int) (string, error) {
	key := getListKey(list)
	//godog.Debug("[GetListPop] key: %s", key)

	value, err := cache.RedisHandle.RPop(key)
	if err != nil {
		godog.Error("[GetListPop] RPop occur error: %s", err)
		return "", err
	}

	return string(value), nil
}

func SetListLPush(list int, value string) error {
	key := getListKey(list)
	godog.Debug("[SetListLPush] key: %s", key)

	if _, err := cache.RedisHandle.LPush(key, value); err != nil {
		godog.Error("[SetListLPush] LPush occur error: %s", err)
		return err
	}

	return nil
}
