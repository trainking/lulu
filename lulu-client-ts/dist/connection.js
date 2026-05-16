"use strict";
/**
 * Connection abstraction for lulu transports.
 *
 * Supports:
 * - WebSocket (browser and Node.js)
 * - TCP (Node.js only, via the `net` module)
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.TcpConnection = exports.WebSocketConnection = void 0;
const packet_1 = require("./packet");
/**
 * Streaming buffer that accumulates raw bytes and emits complete lulu packets.
 */
class PacketStream {
    constructor() {
        this.buffer = new Uint8Array(0);
        this.onPacket = null;
        this.onError = null;
    }
    setHandlers(onPacket, onError) {
        this.onPacket = onPacket;
        this.onError = onError;
    }
    feed(data) {
        // Append new data
        const combined = new Uint8Array(this.buffer.length + data.length);
        combined.set(this.buffer, 0);
        combined.set(data, this.buffer.length);
        this.buffer = combined;
        // Parse all complete packets
        while (this.buffer.length > 0) {
            try {
                const result = (0, packet_1.decodePacket)(this.buffer);
                if (!result)
                    break; // incomplete, wait for more data
                this.onPacket?.(result.packet);
                this.buffer = this.buffer.slice(result.consumed);
            }
            catch (err) {
                this.onError?.(err instanceof Error ? err : new Error(String(err)));
                this.buffer = new Uint8Array(0);
                return;
            }
        }
        // Safety valve: prevent unbounded buffer growth
        if (this.buffer.length > packet_1.MAX_PACKET_SIZE * 2) {
            this.buffer = new Uint8Array(0);
            this.onError?.(new Error("packet stream buffer overflow"));
        }
    }
}
/**
 * WebSocket connection.
 * Works in both browser and Node.js environments.
 */
class WebSocketConnection {
    constructor(config) {
        this.ws = null;
        this.closed = false;
        this.stream = new PacketStream();
        this._errorCb = null;
        const path = config.wsUpgradePath || "/ws";
        const scheme = config.tls ? "wss" : "ws";
        const url = `${scheme}://${config.addr}${path}`;
        this.ws = new WebSocket(url);
        this.ws.binaryType = "arraybuffer";
        this.connectPromise = new Promise((resolve, reject) => {
            if (!this.ws)
                return;
            this.ws.onopen = () => resolve();
            this.ws.onerror = () => {
                reject(new Error(`WebSocket connection failed to ${url}`));
            };
            this.ws.onmessage = (event) => {
                if (event.data instanceof ArrayBuffer) {
                    this.stream.feed(new Uint8Array(event.data));
                }
            };
            this.ws.onclose = () => {
                this.closed = true;
            };
            this.ws.onerror = () => {
                // Errors during active connection are handled via onClose callbacks
            };
        });
    }
    async ready() {
        return this.connectPromise;
    }
    async send(data) {
        if (!this.ws || this.closed) {
            throw new Error("connection closed");
        }
        this.ws.send(data.buffer);
    }
    onMessage(cb) {
        this.stream.setHandlers(cb, this._errorCb ? (e) => this._errorCb(e) : () => { });
    }
    onClose(cb) {
        if (!this.ws)
            return;
        this.ws.onclose = (event) => {
            this.closed = true;
            const wsEvent = event;
            const err = wsEvent.code !== undefined && wsEvent.code !== 1000
                ? new Error(`WebSocket closed: code=${wsEvent.code} reason=${wsEvent.reason}`)
                : undefined;
            cb(err);
        };
    }
    onError(cb) {
        this._errorCb = cb;
        this.stream.setHandlers(this.stream["onPacket"] ? (p) => this.stream["onPacket"]?.(p) : () => { }, cb);
        if (this.ws) {
            this.ws.onerror = () => cb(new Error("WebSocket error"));
        }
    }
    close() {
        if (this.ws && !this.closed) {
            this.closed = true;
            this.ws.close(1000);
        }
    }
}
exports.WebSocketConnection = WebSocketConnection;
/**
 * TCP connection.
 * Only available in Node.js environments (uses the `net` module).
 */
class TcpConnection {
    constructor(config) {
        this.socket = null;
        this.closed = false;
        this.stream = new PacketStream();
        this._errorCb = null;
        // eslint-disable-next-line @typescript-eslint/no-var-requires
        const net = require("net");
        const [host, portStr] = config.addr.split(":");
        const port = parseInt(portStr, 10);
        if (isNaN(port)) {
            throw new Error(`invalid address: ${config.addr} (expected "host:port")`);
        }
        this.connectPromise = new Promise((resolve, reject) => {
            const socket = new net.Socket();
            this.socket = socket;
            socket.connect(port, host, () => resolve());
            socket.on("data", (chunk) => {
                this.stream.feed(new Uint8Array(chunk));
            });
            socket.on("close", () => {
                this.closed = true;
            });
            socket.on("error", (err) => {
                if (!this.closed && this._errorCb) {
                    this._errorCb(err);
                }
                reject(err);
            });
            if (config.readTimeout && config.readTimeout > 0) {
                socket.setTimeout(config.readTimeout * 1000);
            }
        });
    }
    async ready() {
        return this.connectPromise;
    }
    async send(data) {
        if (!this.socket || this.closed) {
            throw new Error("connection closed");
        }
        return new Promise((resolve, reject) => {
            this.socket.write(Buffer.from(data), (err) => {
                if (err)
                    reject(err);
                else
                    resolve();
            });
        });
    }
    onMessage(cb) {
        this.stream.setHandlers(cb, this._errorCb ? (e) => this._errorCb(e) : () => { });
    }
    onClose(cb) {
        if (!this.socket)
            return;
        this.socket.on("close", () => {
            this.closed = true;
            cb();
        });
    }
    onError(cb) {
        this._errorCb = cb;
    }
    close() {
        if (this.socket && !this.closed) {
            this.closed = true;
            this.socket.destroy();
        }
    }
}
exports.TcpConnection = TcpConnection;
