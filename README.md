# Statsd转发器Go版本

设计目的是为了更好的控制statsd发过来的内容，选择性过滤，处理，分发等。

## design

设计思路是类似支持多根管子，内部为一个流水线系统。

过滤器的处理顺序为，输出包统一走以下流程

1. 走重写（如果命中的话），后续过滤均为重写后的值
2. 白名单，命中的将直接输出，跳过黑名单
3. 黑名单，命中的将丢弃

**具体的设计思路请参考代码中的注释**。

## config

```yaml
---
# 过滤器
filters:
  rewrite:   # 转写，支持占位符
    - regexp: deprecated.(.*)
      replace: newspace.\1
  whitelist: # 白名单，命中后直接输出；未命中发给黑名单处理
    - special\..*
  blacklist: # 黑名单，命中的将丢弃
    - old\..*

# 命中黑名单时，打印被过滤的内容
blacklist_report: true

# 输入
input:
  - type: udp
    host: 127.0.0.1
    port: 8125
  - type: udp
    host: 192.168.1.1
    port: 8125

# 输出
output:
  - type: stdout
  - type: udp
    host: 127.0.0.1
    port: 8127
```

## build

```bash
$ cd $GOPATH/src
$ git clone https://github.com/aleiphoenix/statsd-relay-go
$ cd $GOPATH
$ go get gopkg.in/yaml.v2
$ go build src/statsd-relay-go/statsd_relay/cmd/statsd-relay
```

## run

```
./statsd-relay -config </path/to/config.yml>
./statsd-relay -help
```

## signal

* HUP: 在输出打印几个管道的长度情况
