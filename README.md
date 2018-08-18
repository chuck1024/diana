# diana
## Author

```
author: Chuck1024
email : chuck.ch1024@outlook.com
```

## 介绍

### 使用的包
>* [godog](https://github.com/chuck1024/godog)
>* [go-zookeeper](https://github.com/samuel/go-zookeeper/)

### 背景
```
服务A会接收服务B产生的流水，根据流水类型，将相关数据插入数据库或者删除数据；考虑到效率，数据库的操作均是异步操作。
服务A有多个实例运行，例如有A1和A2。
正常情况：
    1、A收到类型为插入db的流水（data1），A将数据插入数据库；
    2、A收到类型为删除db的流水（data1），A将数据删除；
但当流水的数据量较大时，A1还未完成data1相关的数据的插入操作，A2又接收到删除data1的流水2，又开始处理删除data1相关的数据。这样就会产生冲突，数据不一致。
```

### 问题
```
1、保证数据操作的时序性，data1的插入操作先发生，只有处理完成，再开始data1的删除操作；
2、多个实例怎么保证同一个数据由同一个实例处理。
```

### diana方案设计
```
1、保证数据操作的时序性，将A接收到的流水根据流水中的字段（如用户id）取模到redis的list中，然后依次取出进行处理。
    （1）根据整个服务的评估，设置n个list；
    （2）将流水中的字段（如用户id）取模，然后插入相对应的list；
    （3）判断list的长度是否大于0，大于0代表有流水需要处理，然后处理流水。
2、多个实例保证同一个数据由同一个实例处理：
单个实例的实现：
    （1）设置n个协程，一个协程处理一个list；
    （2）协程先去redis加锁，表示某个list已有协程占有，设置过期时间并协程中定期进行过期时间更新（当协程挂掉时，redis中的锁能够及时释放）；
    （3）开始从list中取出流水进行相关数据的处理。
多个实例的实现：
    （1）在zk设置协程的数量和最大实例数，因为一个实例至少拥有一个协程；
    （2）当实例在zk上注册后，根据当前实例总数除以协程数，为一个实例需要开启的协程数；如果有余数，协程现在redis中抢锁，然后在zk的extern目录下注册。
    （3）实例数量增加或减少时，根据计算后的协程数与之前实例中运行的实例数做差，关闭或增加相应数量的协程，并删除在redis中的锁；
```

### 代码实现
```
项目结构
    .
    ├── LICENSE
    ├── README.md
    ├── cache
    │   └── redis.go
    ├── conf
    │   └── conf.json
    ├── main.go
    └── service
        ├── serivce.go
        └── zk.go
> cache为redis的相关操作
> conf为配置文件
> service为项目核心代码
重点：zk.go
ZkData为zk获取的相关数据
    type ZkData struct {
        List     uint64 `json:"list"`     // redis list number
        MaxIdle  uint64 `json:"maxIdle"`  // max idle
        Children uint64 `json:"children"` // children number
    }
func connectZk(zkHost string) 主要实现实例zk的连接、注册等操作
func watch() 实现zk节点变化的通知
func manager() 实现实例中协程的管理
func work() 实现协程的所需要的操作
```

>* 当然zk所有实现的功能也可以用redis实现。嘿嘿嘿...