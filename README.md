# lulu

[中文README](./REAEMD_zh.md)

lulu is a game server development framework. The name comes from the LOL hero - Lulu.

## Features

* Supports multiple protocols: tcp/kcp/websocket
* Modularization
* Easy to use

## Quick Start

Lulu needs a configuration file to configure some specific things about the server. By default, it is stored in'configs/lulu.yaml 'under the project root path. Its configuration is as follows:

```yaml
# The following is the game configuration, that is, app. AppConfig read
# version number
Version: "0.0.1"
# Listening address
Address: "127.0.0.1:8007"
# Transport Layer Protocol，tcp, kcp，websocket
Network: "websocket"
# Upgrade path for websocket
WebsocketPath: "/ws"
# KCP mode, nomarl normal mode fast Quick Mode; default Quick Mode
KcpMode: "fast"
# Read timeout per connection (equal to client side heartbeat timeout) in seconds
ConnReadTimeout: 10
# Write timeout per connection, in seconds
ConnWriteTimeout: 5
# Maximum number of connections
ConnMax: 1000
# After the connection is successful, how long does it take to verify the identity, then it will be disconnected in seconds
ValidTimeout: 10
# Heartbeat limit, the number that cannot be exceeded per minute
HeartLimit: 100
# Password used for service encryption
Password: "66014775009e4106"
# Url for external access
OutUrl: "ws://127.0.0.1:8007"
```

