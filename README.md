##Cola

基于 [rfc1928](https://www.ietf.org/rfc/rfc1928.txt) 和 [rfc1929](https://tools.ietf.org/html/rfc1929) 做的一个 socks5 server。
就着可乐写的，想不出什么名字，所以就叫它 cola 吧。

##使用&测试

```
go run cola.go -c="your_config_file.json"
```

```
curl -v --connect-timeout 5 --socks5 localhost:1080 www.baidu.com
```

##限制
1. 未支持 BIND
2. 认证方式为 NO AUTHENTICATION 和 USERNAME/PASSWORD

##TODO

1. 一个 golang 版的 client。因为 cola 实现了 USERNAME/PASSWORD 的认证方式，
而 [Proxy SwitchyOmega](https://chrome.google.com/webstore/detail/proxy-switchyomega/padekgcemlokbadohgkifijomclgjgif)
似乎并不支持，是不是我打开的方式不对 :( 

2. 因为目前 clear text 方式，所以 USERNAME/PASSWORD 还是太过单薄了，考虑支持 TLS