##About

Since some socks5 clients doesn't support the USERNAME/PASSWORD method, so this client works between the "bare-auth"
client and the server.

First it will negotiate with the "bare-auth" client, then it will negotiate with the server by "username/password auth",
if all that are successful then starts send the data from "bare-auth" client to the server, and vice versa.