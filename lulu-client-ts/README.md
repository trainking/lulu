# lulu-client

TypeScript client SDK for the [lulu](https://github.com/trainking/lulu) game server framework.

## Protocol

The lulu wire protocol uses a fixed 4-byte big-endian header followed by a protobuf-encoded body:

```
┌──────────┬──────────┬─────────────┐
│ Body Len │  Opcode  │    Body     │
│ uint16   │ uint16   │   bytes     │
│ 2 bytes  │ 2 bytes  │ len(Body)   │
└──────────┴──────────┴─────────────┘
```

## Installation

```bash
npm install @trainking/lulu-client
```

## Quick Start

```typescript
import { LuluClient } from "@trainking/lulu-client";

const client = new LuluClient({
  addr: "127.0.0.1:8007",
  network: "websocket",         // or "tcp" (Node.js only)
  wsUpgradePath: "/ws",         // default: "/ws"
  heartbeatOpcode: 0,           // set to enable keep-alive pings
  heartbeatInterval: 30,        // seconds, default: 30
});

// Listen for events
client.on("connect", () => {
  console.log("connected");
  // Send a login request (protobuf-encoded bytes)
  // client.send(1001, loginReqBytes);
});

client.on("message", (packet) => {
  console.log("received opcode:", packet.opcode);
  console.log("received body:", packet.body);
  // Decode with your protobuf library:
  // const msg = YourMessage.decode(packet.body);
});

client.on("close", (err) => {
  console.log("disconnected", err);
});

client.on("error", (err) => {
  console.error("error:", err);
});

// Connect
await client.connect();

// Send a message
const body = new Uint8Array([/* protobuf-encoded data */]);
await client.send(1001, body);

// Disconnect
client.close();
```

## API

### `LuluClient`

#### Constructor

```typescript
new LuluClient(config: ClientConfig)
```

#### Methods

| Method | Description |
|--------|-------------|
| `connect()` | Connect to the server. Returns `Promise<void>`. |
| `send(opcode, body)` | Send a message. `opcode` is 0–65535, `body` is `Uint8Array`. |
| `close()` | Close the connection. |
| `on(event, listener)` | Register an event listener. Returns `this` for chaining. |
| `off(event, listener)` | Remove an event listener. Returns `this` for chaining. |
| `isConnected` | Property: `true` if connected. |

#### Events

| Event | Signature | Description |
|-------|-----------|-------------|
| `connect` | `() => void` | Connection established |
| `message` | `(packet: Packet) => void` | A complete message received |
| `close` | `(error?: Error) => void` | Connection closed |
| `error` | `(error: Error) => void` | Fatal error occurred |

### `ClientConfig`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `addr` | `string` | **required** | Server address, e.g. `"127.0.0.1:8007"` |
| `network` | `"websocket" \| "tcp"` | **required** | Transport protocol |
| `wsUpgradePath` | `string` | `"/ws"` | WebSocket upgrade path |
| `tls` | `boolean` | `false` | Enable TLS (`wss://` / TLS socket) |
| `readTimeout` | `number` | `0` | Read timeout in seconds (0 = none) |
| `writeTimeout` | `number` | `0` | Write timeout in seconds (0 = none) |
| `heartbeatOpcode` | `number` | `0` | Opcode for heartbeat pings (0 = disabled) |
| `heartbeatInterval` | `number` | `30` | Heartbeat interval in seconds |

### `Packet`

| Field | Type | Description |
|-------|------|-------------|
| `opcode` | `number` | Message opcode |
| `body` | `Uint8Array` | Message body (protobuf-encoded) |

### Low-level utilities

```typescript
import { encodePacket, decodePacket, encodeHeartbeat, MAX_PACKET_SIZE } from "@trainking/lulu-client";

// Encode opcode + body into a wire-format buffer
const buf: Uint8Array = encodePacket(1001, bodyBytes);

// Decode a complete packet from a byte stream
const result = decodePacket(rawBytes);
if (result) {
  console.log(result.packet.opcode, result.packet.body);
  console.log(result.consumed); // bytes consumed
}

// Create a heartbeat packet
const hb: Uint8Array = encodeHeartbeat(0);

// Maximum packet body size (65,535 bytes)
console.log(MAX_PACKET_SIZE);
```

## Using with Protobuf

This SDK does **not** bundle a protobuf library — you bring your own. The body bytes in `send()` and `Packet.body` are raw `Uint8Array` values that you encode/decode with your protobuf library of choice:

**With `protobufjs`:**
```typescript
import * as protobuf from "protobufjs";

const root = await protobuf.load("messages.proto");
const LoginReq = root.lookupType("msg.LoginReq");

const body = LoginReq.encode({ username: "test" }).finish();
await client.send(1001, body);
```

**With `@bufbuild/protobuf`:**
```typescript
import { YourMessage } from "./gen/messages_pb";

const body = new YourMessage({ username: "test" }).toBinary();
await client.send(1001, body);
```
