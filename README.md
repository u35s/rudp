[![Build Status](https://travis-ci.org/u35s/rudp.svg?branch=master)](https://travis-ci.org/u35s/rudp)
[![Coverage Status](https://coveralls.io/repos/github/u35s/rudp/badge.svg)](https://coveralls.io/github/u35s/rudp)

# rudp
rudp采用请求回应机制,实现了UDP的可靠传输,即接收方检查是否丢失数据,然后向发送方请求丢失的数据,因此发送方必须保留已经发送过的数据一定时间来回应数据丢失。为了减小发送方数据保留量,在每收到n个包时通知发送方n之前的包已经收到可以清除了,另外超过设定的包超时时间后也会清除。

# 使用
1 创建rudp对象

```golang
rudp := rudp.New()
```

2 发送消息,n 发送的的消息长度,err 是否出错

```golang
n ,err := rudp.Send(bts []byte)
```

3 接受消息,n 返回接受到的的消息长度,err 是否出错

```golang
n , err := rudp.Recv(data []byte)
```

4 更新时间获取要发送的消息,如果设置的sendDelay大于更新tick,update返回nil,下次调用时间到时会返回所有的消息链表

```golang
var package *Package = rudp.Update(tick int)
```
5 相关设置

```golang
rudp.SetCorruptTick(n int)    //设置超过n个tick连接丢失
rudp.SetExpiredTick(n int)    //设置发送的消息最大保留n个tick
rudp.SetSendDelayTick(n int)  //设置n个tick发送一次消息包
rudp.SetMissingTime(n int)    //设置n纳秒没有收到消息包就认为消息丢失，请求重发
```

# 兼容tcp
另外rudp也实现了tcp的相关接口,很容易改造现有的tcp项目为rudp

### 服务端

1 监听udp端口

```golang
addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 9981}
conn, err := net.ListenUDP("udp", addr)
if err != nil {
	fmt.Println(err)
	return
}
```
2 接受连接

```golang
listener := rudp.NewListener(conn)
rconn, err := listener.AcceptRudp()
if err != nil {
	fmt.Printf("accept err %v\n", err)
	return
}
```
3 读取消息

```golang
data := make([]byte, rudp.MAX_PACKAGE)
n, err := rconn.Read(data)
if err != nil {
	fmt.Printf("read err %s\n", err)
	return
}
```
4 发送消息

```golang
n , err := rconn.Write([]byte("hello rudp"))
```

### 客户端

1 拨号

```golang
raddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9981}
//raddr := net.UDPAddr{IP: net.ParseIP("47.89.180.105"), Port: 9981}
laddr := net.UDPAddr{IP: net.IPv4zero, Port: 0}
conn, err := net.DialUDP("udp", &laddr, &raddr)
if err != nil {
	fmt.Println(err)
	return
}
```
2 创建conn

```golang
rconn := rudp.NewConn(conn, rudp.New())
```
3 发送消息,同服务端
4 接受消息,同服务端

### 相关设置

```golang
rudp.SetAtuoSend(bool) 设置rudp是否自动发送消息
rudp.SetSendTick() 设置发送的间隔(为0时自动发送消息不启用)
rudp.SetMaxSendNumPerTick() 设置每个tick可以最大发送的消息数量
``` 

# Links
1. https://github.com/cloudwu/rudp --rudp in c
2. https://blog.codingnow.com/2016/03/reliable_udp.html --blog of rudp
