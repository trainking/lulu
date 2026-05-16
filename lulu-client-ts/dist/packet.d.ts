/**
 * Packet encoding/decoding for the lulu binary protocol.
 *
 * Protocol format (big-endian):
 * ┌──────────┬──────────┬─────────────┐
 * │ Body Len │  Opcode  │    Body     │
 * │ uint16   │ uint16   │   bytes     │
 * │ 2 bytes  │ 2 bytes  │ len(Body)   │
 * └──────────┴──────────┴─────────────┘
 */
import type { Packet } from "./types";
/** Maximum packet body size (64 MB), same as server-side MaxPacketSize */
export declare const MAX_PACKET_SIZE: number;
/**
 * Encode an opcode and body into a lulu protocol buffer.
 */
export declare function encodePacket(opcode: number, body: Uint8Array): Uint8Array;
/**
 * Try to decode a lulu protocol packet from raw bytes.
 *
 * Returns the decoded packet and the number of bytes consumed,
 * or null if the buffer doesn't contain a complete packet yet.
 */
export declare function decodePacket(data: Uint8Array): {
    packet: Packet;
    consumed: number;
} | null;
/**
 * Create a heartbeat packet (opcode only, empty body).
 */
export declare function encodeHeartbeat(opcode: number): Uint8Array;
