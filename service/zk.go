/**
 * Copyright 2018 Diana Author. All rights reserved.
 * Author: Chuck1024
 */

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chuck1024/diana/dao/cache"
	"github.com/chuck1024/godog"
	"github.com/chuck1024/godog/utils"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

type ZkData struct {
	SortSetNum uint64 `json:"sortSetNum"` // redis SortSet number
	MaxIdle    uint64 `json:"maxIdle"`    // max idle
	Children   uint64 `json:"children"`   // children number
}

var (
	sortSetChan = make(chan *ZkData, 100)
	stopChan    = make(chan bool, 100)
	errChan     = make(chan bool, 100)

	initRoutines uint64 = 0
	Conn         *zk.Conn

	rootPath, _ = godog.AppConfig.String("rootPath")
)

func isExistRoot() (err error) {
	isExist, _, err := Conn.Exists(rootPath)
	if err != nil {
		godog.Error("[isExistRoot] Exists occur error: %s", err)
		return
	}

	SortSetNum, _ := godog.AppConfig.Int("sortSetNum")
	maxIdle, _ := godog.AppConfig.Int("maxIdle")
	if !isExist {
		data := &ZkData{
			SortSetNum: uint64(SortSetNum),
			MaxIdle:    uint64(maxIdle),
		}

		dataByte, _ := json.Marshal(data)

		path, err := Conn.Create(rootPath, dataByte, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			godog.Error("[isExistRoot] create rootPath occur error: %s", err)
			return err
		}

		if rootPath != path {
			godog.Error("[isExistRoot] create rootPath [%s] != path [%s]", rootPath, path)
			return errors.New("rootPath is equal path")
		}
	}

	is, _, err := Conn.Exists(rootPath + "/extern")
	if err != nil {
		godog.Error("[isExistRoot] Exists rootPath/extern/ occur error:%s", err)
		return
	}

	if !is {
		path, err := Conn.Create(rootPath+"/extern", nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			godog.Error("[isExistRoot] create rootPath occur error: %s", err)
			return err
		}

		if rootPath+"/extern" != path {
			godog.Error("[isExistRoot] create rootPath [%s + /extern] != path [%s]", rootPath, path)
			return errors.New("rootPath/extern is equal path")
		}
	}

	return
}

func connectZk(zkHost string) (err error) {
	var hosts = []string{zkHost}
	conn, _, err := zk.Connect(hosts, time.Second*5, zk.WithLogInfo(false))
	if err != nil {
		godog.Error("[connectZk] zk connect occur error:%s", err)
		return
	}

	Conn = conn
	//defer conn.Close()
	if err := isExistRoot(); err != nil {
		godog.Error("[connectZk] isExistRoot occur error:%s", err)
		return err
	}

	p := rootPath + "/" + utils.GetLocalIP()
	path, err := Conn.Create(p, nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err == nil {
		if path == p {
			godog.Debug("[connectZk] connect success!")
		} else {
			return
		}
	}

	Byte, _, err := Conn.Get(rootPath)
	if err != nil {
		godog.Error("[connectZk] get occur error:%s", err)
		return
	}

	t := &ZkData{}
	if err := json.Unmarshal(Byte, &t); err != nil {
		godog.Error("[connectZk] json unmarshal occur error:%s", err)
		return err
	}

	children, _, err := conn.Children(rootPath)
	if err != nil {
		godog.Error("[connectZk] get children occur error:%s", err)
		return
	}

	t.Children = uint64(len(children) - 1)
	sortSetChan <- t

	godog.Debug("[connectZk] root:%v, children:%d", *t, len(children))

	return
}

func watch() {
	for {
		_, _, childCh, err := Conn.ChildrenW(rootPath)
		if err != nil {
			godog.Error("[watch] children watch occur error:%s", err)
			continue
		}

		select {
		case childEvent := <-childCh:
			if childEvent.Type == zk.EventNodeChildrenChanged {
				godog.Debug("[watch] receive znode children changed event:%d", childEvent.Type)

				Byte, _, err := Conn.Get(rootPath)
				if err != nil {
					godog.Error("[watch] get path data occur error:%s", err)
					continue
				}

				t := &ZkData{}
				if err := json.Unmarshal(Byte, &t); err != nil {
					godog.Error("[watch] json unmarshal occur error:%s", err)
					continue
				}

				children, _, err := Conn.Children(rootPath)
				if err != nil {
					godog.Error("[watch] get children occur error:%s", err)
					return
				}

				t.Children = uint64(len(children) - 1)
				sortSetChan <- t
				godog.Debug("root:%v ,children:%d", *t, len(children))
			}
		}
	}
}

func manager() {
	for {
		select {
		case t := <-sortSetChan:
			godog.Debug("[manager] sortSetChan:%v", *t)
			r := 0
			routines := t.SortSetNum / t.Children
			godog.Debug("[manager] initRoutines: %d , routines: %d ", initRoutines, routines)

			if initRoutines == 0 {
				r = int(routines)
			} else {
				if initRoutines < routines {
					r = int(routines - initRoutines)
				} else {
					r = int(initRoutines - routines)
				}
			}

			godog.Debug("[manager] r:%d", r)
			count := 0
			if initRoutines == 0 || initRoutines <= routines {
				for i := 0; i < r; i++ {
					c := getLock(t.SortSetNum)
					count += c
					time.Sleep(10 * time.Millisecond)
				}

				if r-count > 0 {
					for {
						c := getLock(t.SortSetNum)
						count += c
						if (r - count) == 0 {
							break
						}
						time.Sleep(10 * time.Millisecond)
					}
				}

			} else if initRoutines > routines {
				for i := 0; i < r; i++ {
					godog.Debug("[manager] stopChan<-true :%d", i)
					stopChan <- true
				}
			}

			y := t.SortSetNum % t.Children
			if y != 0 {
				for i := 0; i < int(y); i++ {
					p := fmt.Sprintf("%s/extern/%d", rootPath, i)
					path, err := Conn.Create(p, []byte(utils.GetLocalIP()), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
					if err == nil {
						if p == path {
							getLock(t.SortSetNum)
						}
					}
				}
			}

			go func(sortSets uint64) {
				for {
					select {
					case <-errChan:
						time.Sleep(5 * time.Second)
						getLock(sortSets)
					default:
					}
					time.Sleep(10 * time.Millisecond)
				}
			}(t.SortSetNum)

			initRoutines = routines
		}
	}
}

func getLock(sortSets uint64) int {
	var f int
	for i := 0; i < int(sortSets); i++ {
		if _, err := cache.SetLock(i); err != nil {
			continue
		}

		f = i
		godog.Debug("[getLock] f: %d", f)
		go func(v int) {
			work(v)
		}(f)
		return 1
	}

	return 0
}

func work(f int) {
	defer func() {
		if r := recover(); r != nil {
			godog.Error("[work] work sortSet:%d occur error:%s", f, r)
			errChan <- true
		}
	}()

	stop := make(chan bool)
	t := time.NewTicker(2 * time.Second)
	go func(i int) {
		for {
			select {
			case <-t.C:
				if err := cache.ExpireLock(i); err != nil {
					continue
				}
			case <-stop:
				cache.DelLock(i)
				return
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}(f)

	for {
		l, err := cache.GetZCard(f)
		if err != nil {
			godog.Error("[work] GetZCard occur error: %s", err)
			continue
		}

		if l == 0 {
			select {
			case <-stopChan:
				godog.Debug("[work] received stop chan")
				stop <- true
				return
			default:
				time.Sleep(10 * time.Millisecond)
			}

			continue
		}

		value, err := cache.GetZRange(f)
		if err != nil {
			godog.Error("[work] cache GetZRange occur error: %s", err)
			continue
		}

		retryNum := 0
	Retry:
		if err = dispatchChanData(value); err != nil {
			retryNum++
			godog.Error("[work] dispatchChanData occur error: %s ", err)

			if retryNum < 3 {
				goto Retry
			}
		}

		if err = cache.DelSortSet(f, value); err != nil {
			godog.Error("[work] DelSortSet occur error: %s ", err)
			continue
		}

		select {
		case <-stopChan:
			godog.Debug("[work] received stop chan %d", f)
			stop <- true
			return
		default:
		}

	}
}
