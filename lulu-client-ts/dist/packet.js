"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.MAX_PACKET_SIZE = void 0;
exports.encodePacket = encodePacket;
exports.decodePacket = decodePacket;
exports.encodeHeartbeat = encodeHeartbeat;
/** Maximum packet body size, same as the server-side uint16 body length header. */
exports.MAX_PACKET_SIZE = 0xffff;
/** Header size in bytes (2 + 2 = 4) */
const HEADER_SIZE = 4;
/**
 * Encode an opcode and body into a lulu protocol buffer.
 */
function encodePacket(opcode, body) {
    if (opcode < 0 || opcode > 0xffff) {
        throw new Error(`opcode out of range: ${opcode} (must be 0-65535)`);
    }
    if (body.length > exports.MAX_PACKET_SIZE) {
        throw new Error(`body too large: ${body.length} bytes (max ${exports.MAX_PACKET_SIZE})`);
    }
    const buf = new Uint8Array(HEADER_SIZE + body.length);
    const view = new DataView(buf.buffer);
    // Big-endian: body length (uint16) at offset 0
    view.setUint16(0, body.length, false);
    // Big-endian: opcode (uint16) at offset 2
    view.setUint16(2, opcode, false);
    // Copy body
    buf.set(body, HEADER_SIZE);
    return buf;
}
/**
 * Try to decode a lulu protocol packet from raw bytes.
 *
 * Returns the decoded packet and the number of bytes consumed,
 * or null if the buffer doesn't contain a complete packet yet.
 */
function decodePacket(data) {
    if (data.length < HEADER_SIZE) {
        return null; // incomplete header
    }
    const view = new DataView(data.buffer, data.byteOffset, data.length);
    const bodyLen = view.getUint16(0, false); // big-endian
    const opcode = view.getUint16(2, false); // big-endian
    if (bodyLen > exports.MAX_PACKET_SIZE) {
        throw new Error(`packet body too large: ${bodyLen} bytes (max ${exports.MAX_PACKET_SIZE})`);
    }
    const totalLen = HEADER_SIZE + bodyLen;
    if (data.length < totalLen) {
        return null; // incomplete body
    }
    const body = data.slice(HEADER_SIZE, totalLen);
    return {
        packet: { opcode, body },
        consumed: totalLen,
    };
}
/**
 * Create a heartbeat packet (opcode only, empty body).
 */
function encodeHeartbeat(opcode) {
    return encodePacket(opcode, new Uint8Array(0));
}
