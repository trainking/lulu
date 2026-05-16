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

export { LuluClient } from "./client";

export {
  encodePacket,
  decodePacket,
  encodeHeartbeat,
  MAX_PACKET_SIZE,
} from "./packet";

export type {
  ClientConfig,
  NetworkType,
  Packet,
  ClientEvents,
} from "./types";
