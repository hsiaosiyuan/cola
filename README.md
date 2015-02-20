##Cola

A socks5 server implements [rfc1928](https://www.ietf.org/rfc/rfc1928.txt) and [rfc1929](https://tools.ietf.org/html/rfc1929).
Feel hard to give it a name but I was writing it with drinking cola, so just call it cola.

##Usage&Test

```
go run cola.go -c="your_config_file.json"
```

```
curl -v --connect-timeout 5 --socks5 localhost:1080 www.baidu.com
```

##Restrictions
1. Haven't support the *BIND* command.
2. Supported authentication	methods are only *NO AUTHENTICATION* and *USERNAME/PASSWORD*.

##TODO
1. To support TLS.