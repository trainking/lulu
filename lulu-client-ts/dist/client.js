"use strict";
/**
 * High-level lulu client SDK.
 *
 * ```typescript
 * import { LuluClient } from "@trainking/lulu-client";
 *
 * const client = new LuluClient({
 *   addr: "127.0.0.1:8007",
 *   network: "websocket",
 * });
 *
 * client.on("connect", () => {
 *   // Send a login request (protobuf-encoded body)
 *   client.send(1001, loginReqBytes);
 * });
 *
 * client.on("message", (packet) => {
 *   console.log("opcode:", packet.opcode);
 *   // decode packet.body as protobuf...
 * });
 *
 * client.on("close", (err) => {
 *   console.log("disconnected", err);
 * });
 *
 * client.on("error", (err) => {
 *   console.error("client error", err);
 * });
 *
 * await client.connect();
 * ```
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.LuluClient = void 0;
const packet_1 = require("./packet");
const connection_1 = require("./connection");
class LuluClient {
    constructor(config) {
        this.conn = null;
        this.listeners = new Map();
        this.heartbeatTimer = null;
        this.connected = false;
        this.config = {
            wsUpgradePath: config.wsUpgradePath ?? "/ws",
            tls: config.tls ?? false,
            readTimeout: config.readTimeout ?? 0,
            writeTimeout: config.writeTimeout ?? 0,
            heartbeatOpcode: config.heartbeatOpcode ?? 0,
            heartbeatInterval: config.heartbeatInterval ?? 30,
            ...config,
        };
    }
    /**
     * Connect to the lulu server.
     */
    async connect() {
        if (this.connected) {
            throw new Error("already connected");
        }
        const conn = this.createConnection();
        await conn.ready();
        conn.onMessage((packet) => {
            this.emit("message", packet);
        });
        conn.onClose((error) => {
            this.connected = false;
            this.stopHeartbeat();
            this.emit("close", error);
        });
        conn.onError((error) => {
            this.emit("error", error);
        });
        this.conn = conn;
        this.connected = true;
        this.emit("connect");
        this.startHeartbeat();
    }
    /**
     * Send a message to the server.
     *
     * @param opcode - The message opcode (0–65535)
     * @param body - The protobuf-encoded message body
     */
    async send(opcode, body) {
        if (!this.conn || !this.connected) {
            throw new Error("not connected");
        }
        const raw = (0, packet_1.encodePacket)(opcode, body);
        await this.conn.send(raw);
    }
    /**
     * Register an event listener.
     */
    on(event, listener) {
        const existing = this.listeners.get(event) || [];
        existing.push(listener);
        this.listeners.set(event, existing);
        return this;
    }
    /**
     * Remove an event listener.
     */
    off(event, listener) {
        const existing = this.listeners.get(event) || [];
        this.listeners.set(event, existing.filter((l) => l !== listener));
        return this;
    }
    /**
     * Close the connection.
     */
    close() {
        this.stopHeartbeat();
        if (this.conn) {
            this.conn.close();
            this.conn = null;
        }
        this.connected = false;
    }
    get isConnected() {
        return this.connected;
    }
    // ── private ──────────────────────────────────────────────
    createConnection() {
        switch (this.config.network) {
            case "websocket":
                return new connection_1.WebSocketConnection(this.config);
            case "tcp":
                return new connection_1.TcpConnection(this.config);
            default:
                throw new Error(`unsupported network type: "${this.config.network}"`);
        }
    }
    startHeartbeat() {
        if (!this.config.heartbeatOpcode)
            return;
        const interval = (this.config.heartbeatInterval || 30) * 1000;
        this.heartbeatTimer = setInterval(async () => {
            try {
                const hb = (0, packet_1.encodeHeartbeat)(this.config.heartbeatOpcode);
                await this.conn?.send(hb);
            }
            catch {
                this.close();
            }
        }, interval);
    }
    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }
    emit(event, ...args) {
        const listeners = this.listeners.get(event) || [];
        for (const listener of listeners) {
            try {
                listener(...args);
            }
            catch (err) {
                console.error(`lulu-client: error in "${event}" listener:`, err);
            }
        }
    }
}
exports.LuluClient = LuluClient;
