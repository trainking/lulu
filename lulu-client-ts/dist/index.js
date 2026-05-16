"use strict";
/**
 * lulu-client — TypeScript client SDK for the lulu game server framework.
 *
 * Protocol: fixed 4-byte big-endian header (2 bytes body length + 2 bytes opcode)
 *           followed by a protobuf-encoded variable-length body.
 *
 * @example
 * ```typescript
 * import { LuluClient, encodePacket, decodePacket } from "@trainking/lulu-client";
 *
 * const client = new LuluClient({ addr: "127.0.0.1:8007", network: "websocket" });
 * await client.connect();
 * client.send(1001, protobufEncodedBody);
 * ```
 *
 * @packageDocumentation
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.MAX_PACKET_SIZE = exports.encodeHeartbeat = exports.decodePacket = exports.encodePacket = exports.LuluClient = void 0;
var client_1 = require("./client");
Object.defineProperty(exports, "LuluClient", { enumerable: true, get: function () { return client_1.LuluClient; } });
var packet_1 = require("./packet");
Object.defineProperty(exports, "encodePacket", { enumerable: true, get: function () { return packet_1.encodePacket; } });
Object.defineProperty(exports, "decodePacket", { enumerable: true, get: function () { return packet_1.decodePacket; } });
Object.defineProperty(exports, "encodeHeartbeat", { enumerable: true, get: function () { return packet_1.encodeHeartbeat; } });
Object.defineProperty(exports, "MAX_PACKET_SIZE", { enumerable: true, get: function () { return packet_1.MAX_PACKET_SIZE; } });
