# diana

diana: 黛安娜（罗马神话中之处女性守护神，狩猎女神和月亮女神，相当于希腊神话中的Artemis）

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
    2、A收到类型为删除db的流水（data1），A将数据删除；(当然这里的删除指的软删除)
但当流水的数据量较大时，A1还未完成data1相关的数据的插入操作，A2又接收到删除data1的流水2，又开始处理删除data1相关的数据。这样就会产生冲突，数据不一致。
```

### 问题
```
1、保证数据操作的时序性，data1的插入操作先发生，只有处理完成，再开始data1的删除操作；
2、多个实例怎么保证同一个数据由同一个实例处理。
```

### diana方案设计
```
1、保证数据操作的时序性，将A接收到的流水根据流水中的字段（如用户id）取模到redis的sortSet中，然后依次取出进行处理。
    （1）根据整个服务的评估，设置n个sortSet；
    （2）将流水中的字段（如用户id）取模，然后插入相对应的sortSet，socre的时间戳，单位纳秒；
    （3）判断sortSet的元素数量是否大于0，大于0代表有流水需要处理，然后取出分数最小的元素处理；
    （4）当处理数据成功后，再到对应的sortSet删除该元素，如果处理出错，将重试3次。
2、多个实例保证同一个数据由同一个实例处理：
单个实例的实现：
    （1）设置n个协程，一个协程处理一个sortSet；
    （2）协程先去redis加锁，表示某个sortSet已有协程占有，设置过期时间并协程中定期进行过期时间更新（当协程挂掉时，redis中的锁能够及时释放）；
    （3）开始从sortSet中最早的member取出，进行相关数据的处理。
多个实例的实现：
    （1）在zk设置协程的数量和最大实例数，因为一个实例至少拥有一个协程；
    （2）当实例在zk上注册后，根据当前实例总数除以协程数，为一个实例需要开启的协程数；如果有余数，协程现在redis中抢锁，然后在zk的extern目录下注册。
    （3）实例数量增加或减少时，根据计算后的协程数与之前实例中运行的实例数做差，关闭或增加相应数量的协程，并删除在redis中的锁；
```

### 项目结构
```
    .
    ├── LICENSE
    ├── README.md
    ├── conf
    │   └── conf.json
    ├── main.go
    ├── model
    │   ├── dao
    │   │   └── cache
    │   │       └── redis_dao.go
    │   └── service
    │       ├── handle_srv.go
    │       ├── serivce_srv.go
    │       └── zk.go
    └── vendor
> cache为redis的相关操作
> conf为配置文件
> service为项目核心代码
重点：zk.go
ZkData为zk获取的相关数据
    type ZkData struct {
        SortSetNum  uint64 `json:"list"`  // redis SortSet number
        MaxIdle  uint64 `json:"maxIdle"`  // max idle
        Children uint64 `json:"children"` // children number
    }
func connectZk(zkHost string) 主要实现实例zk的连接、注册等操作
func watch() 实现zk节点变化的通知
func manager() 实现实例中协程的管理
func work() 实现协程的所需要的操作
func isExistRoot()方法是判断zk是否有rootPath路径，没有就创建rootPath路径和sortSet、maxIdle。当然也可以手动在zk中先创建。
```

### 运行
```
1、开启redis和zk
2、conf/conf.json配置redis、zk地址和rootPath、list、maxIdle等配置项
3、如果没有多台机器，通过修改zk.go 102行和212行中的utils.GetLocalIP()为其他地址，以达到zk的路径下注册不同的节点。注：该方法是获取本机地址

实验配置conf.json：
{
  "Log": {
    "File": "log/diana.log",
    "Level": "debug",
    "Daemon": false,
    "Suffix": "20060102"
  },
  "Prog": {
    "CPU": 0,
    "HealthPort": 0
  },
  "Server": {
    "AppName": "diana",
    "HttpPort":0,
    "TcpPort": 0
  },

  "zkHost":"127.0.0.1:2181",
  "redis" : "tcp://127.0.0.1:6379/0?cluster=false&idleTimeout=1s&maxIdle=10&madActive=200",

  "rootPath": "/diana",
  "sortSetNum":20,
  "maxIdle":20
}

实验步骤：
1）启动第一个实例：
>$ go run main.go
[2018/08/19 13:17:48.478] INFO  [redis.go:32] [InitRedis] redis conn ok: tcp://127.0.0.1:6379/0?timeout=60s&maxidle=10
[2018/08/19 13:17:48.500] DEBUG [zk.go:106] [connectZk] connect success!
[2018/08/19 13:17:48.501] DEBUG [zk.go:133] [connectZk] root:{20 20 1}, children:2
[2018/08/19 13:17:48.501] INFO  [godog.go:47] [App.Run] start
[2018/08/19 13:17:48.501] DEBUG [zk.go:181] [manager] listChan:{20 20 1}
[2018/08/19 13:17:48.501] INFO  [signal.go:44] [Signal] register signal ok
[2018/08/19 13:17:48.501] DEBUG [zk.go:184] [manager] initRoutines: 0 , routines: 20
[2018/08/19 13:17:48.501] DEBUG [zk.go:196] [manager] r:20
[2018/08/19 13:17:48.502] INFO  [httpserver.go:81] [Run] No http Serve port for application
[2018/08/19 13:17:48.502] INFO  [godog.go:80] [App.Run] Hasn't http server port
[2018/08/19 13:17:48.502] INFO  [tcpserver.go:52] [Run] no tcp serve port
[2018/08/19 13:17:48.502] INFO  [godog.go:90] [App.Run] Hasn't tcp server port
[2018/08/19 13:17:48.502] DEBUG [zk.go:244] [getLock] f: 0
[2018/08/19 13:17:48.502] DEBUG [zk.go:244] [getLock] f: 1
[2018/08/19 13:17:48.503] DEBUG [zk.go:244] [getLock] f: 2
[2018/08/19 13:17:48.504] DEBUG [zk.go:244] [getLock] f: 3
[2018/08/19 13:17:48.505] DEBUG [zk.go:244] [getLock] f: 4
[2018/08/19 13:17:48.506] DEBUG [zk.go:244] [getLock] f: 5
[2018/08/19 13:17:48.507] DEBUG [zk.go:244] [getLock] f: 6
[2018/08/19 13:17:48.509] DEBUG [zk.go:244] [getLock] f: 7
[2018/08/19 13:17:48.511] DEBUG [zk.go:244] [getLock] f: 8
[2018/08/19 13:17:48.513] DEBUG [zk.go:244] [getLock] f: 9
[2018/08/19 13:17:48.515] DEBUG [zk.go:244] [getLock] f: 10
[2018/08/19 13:17:48.518] DEBUG [zk.go:244] [getLock] f: 11
[2018/08/19 13:17:48.527] DEBUG [zk.go:244] [getLock] f: 12
[2018/08/19 13:17:48.533] DEBUG [zk.go:244] [getLock] f: 13
[2018/08/19 13:17:48.539] DEBUG [zk.go:244] [getLock] f: 14
[2018/08/19 13:17:48.543] DEBUG [zk.go:244] [getLock] f: 15
[2018/08/19 13:17:48.549] DEBUG [zk.go:244] [getLock] f: 16
[2018/08/19 13:17:48.553] DEBUG [zk.go:244] [getLock] f: 17
[2018/08/19 13:17:48.560] DEBUG [zk.go:244] [getLock] f: 18
[2018/08/19 13:17:48.566] DEBUG [zk.go:244] [getLock] f: 19

2）修改zk.go 102行utils.GetLocalIP() 为"127.0.0.1"，启动第二个实例。manager方法根据znode数量计算协程数量，
我们会发现第二个实例启动了10个协程：
>$ go run main.go
[2018/08/19 13:18:27.197] INFO  [redis.go:32] [InitRedis] redis conn ok: tcp://127.0.0.1:6379/0?timeout=60s&maxidle=10
[2018/08/19 13:18:27.222] DEBUG [zk.go:105] [connectZk] connect success!
[2018/08/19 13:18:27.223] DEBUG [zk.go:132] [connectZk] root:{20 20 2}, children:3
[2018/08/19 13:18:27.223] INFO  [godog.go:47] [App.Run] start
[2018/08/19 13:18:27.223] INFO  [signal.go:44] [Signal] register signal ok
[2018/08/19 13:18:27.223] DEBUG [zk.go:180] [manager] listChan:{20 20 2}
[2018/08/19 13:18:27.224] DEBUG [zk.go:183] [manager] initRoutines: 0 , routines: 10
[2018/08/19 13:18:27.224] DEBUG [zk.go:195] [manager] r:10
[2018/08/19 13:18:27.224] INFO  [httpserver.go:81] [Run] No http Serve port for application
[2018/08/19 13:18:27.224] INFO  [godog.go:80] [App.Run] Hasn't http server port
[2018/08/19 13:18:27.224] INFO  [tcpserver.go:52] [Run] no tcp serve port
[2018/08/19 13:18:27.224] INFO  [godog.go:90] [App.Run] Hasn't tcp server port
[2018/08/19 13:18:27.226] DEBUG [zk.go:243] [getLock] f: 1
[2018/08/19 13:18:27.226] DEBUG [zk.go:243] [getLock] f: 0
[2018/08/19 13:18:27.228] DEBUG [zk.go:243] [getLock] f: 3
[2018/08/19 13:18:27.230] DEBUG [zk.go:243] [getLock] f: 4
[2018/08/19 13:18:27.233] DEBUG [zk.go:243] [getLock] f: 9
[2018/08/19 13:18:27.240] DEBUG [zk.go:243] [getLock] f: 11
[2018/08/19 13:18:27.247] DEBUG [zk.go:243] [getLock] f: 12
[2018/08/19 13:18:27.254] DEBUG [zk.go:243] [getLock] f: 13
[2018/08/19 13:18:27.262] DEBUG [zk.go:243] [getLock] f: 14
[2018/08/19 13:18:27.270] DEBUG [zk.go:243] [getLock] f: 16

此时第一个实例的watch检测到znode数量发生变化，manage方法重新计算每个实例的协程数量，需要关闭10个协程，则随机关闭10个协程，并删除redis占有的
锁。我们通过日志输出时间也可以发现，只有第一个实例在redis释放锁后，第二个实例才能抢到锁，并启动：
[2018/08/19 13:18:27.222] DEBUG [zk.go:149] [watch] receive znode children changed event:4
[2018/08/19 13:18:27.223] DEBUG [zk.go:171] root:{20 20 2} ,children:3
[2018/08/19 13:18:27.223] DEBUG [zk.go:181] [manager] listChan:{20 20 2}
[2018/08/19 13:18:27.224] DEBUG [zk.go:184] [manager] initRoutines: 20 , routines: 10
[2018/08/19 13:18:27.224] DEBUG [zk.go:196] [manager] r:10
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :0
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :1
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :2
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :3
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :4
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :5
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :6
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :7
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :8
[2018/08/19 13:18:27.224] DEBUG [zk.go:203] [manager] stopChan<-true :9
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:4
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:0
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:9
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:12
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:3
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:13
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:11
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:16
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:14
[2018/08/19 13:18:27.225] DEBUG [zk.go:286] [work] received stop chan
[2018/08/19 13:18:27.225] DEBUG [redis.go:57] [DelLock] key: diana:lock:1

3）使用 control+C 关闭第二个实例：
^C[2018/08/19 13:19:04.618] INFO  [signal.go:35] [Signal] receive signal SIGINT or SIGTERM, to stop server...
[2018/08/19 13:19:04.618] INFO  [godog.go:59] [Run] server stop...code: 30
[2018/08/19 13:19:05.621] INFO  [godog.go:61] [Run] server stop...ok

此时第一个实例watch检测到znode减少为一个，manager则计算每个实例的协程数，第一个实例将把第二个实例上的10个协程抢过来：
[2018/08/19 13:19:11.369] DEBUG [zk.go:149] [watch] receive znode children changed event:4
[2018/08/19 13:19:11.370] DEBUG [zk.go:171] root:{20 20 1} ,children:2
[2018/08/19 13:19:11.370] DEBUG [zk.go:181] [manager] listChan:{20 20 1}
[2018/08/19 13:19:11.370] DEBUG [zk.go:184] [manager] initRoutines: 10 , routines: 20
[2018/08/19 13:19:11.370] DEBUG [zk.go:196] [manager] r:10
[2018/08/19 13:19:11.370] DEBUG [zk.go:244] [getLock] f: 0
[2018/08/19 13:19:11.371] DEBUG [zk.go:244] [getLock] f: 1
[2018/08/19 13:19:11.372] DEBUG [zk.go:244] [getLock] f: 3
[2018/08/19 13:19:11.373] DEBUG [zk.go:244] [getLock] f: 4
[2018/08/19 13:19:11.375] DEBUG [zk.go:244] [getLock] f: 9
[2018/08/19 13:19:11.377] DEBUG [zk.go:244] [getLock] f: 11
[2018/08/19 13:19:11.380] DEBUG [zk.go:244] [getLock] f: 12
[2018/08/19 13:19:11.384] DEBUG [zk.go:244] [getLock] f: 13
[2018/08/19 13:19:11.393] DEBUG [zk.go:244] [getLock] f: 14
[2018/08/19 13:19:11.400] DEBUG [zk.go:244] [getLock] f: 16

实验结束
```

### 总结
```
1、第一个实例日志中打印的[zk.go:133] [connectZk] root:{20 20 1}, children:2，children为什么是2？
    因为isExistRoot()会创建一个extern的znode。当sortSetNum除以实例数有余数时，每个实例则现在extern注册一个子节点，代表这个
    sortSet已被某个实例占用。子节点的data为实例的地址，方便查看多余出来的sortSett在哪个实例上运行。
2、读者可以试着运行3个实例的实验。作者已经做过多个实例的情况，在此就不在赘述。
3、后续的优化，可以使用有序集合替代list。--此版本已优化
4、读者有任何问题，都可以发邮件与作者联系讨论。
```

## License
diana is released under the [**MIT LICENSE**](http://opensource.org/licenses/mit-license.php).  
