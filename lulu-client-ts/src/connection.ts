/**
 * Connection abstraction for lulu transports.
 *
 * Supports:
 * - WebSocket (browser and Node.js)
 * - TCP (Node.js only, via the `net` module)
 */

import type { ClientConfig, Packet } from "./types";
import { decodePacket, MAX_PACKET_SIZE } from "./packet";

/** Low-level connection interface */
export interface IConnection {
  /** Wait for the connection to be established */
  ready(): Promise<void>;
  /** Send raw bytes over the connection */
  send(data: Uint8Array): Promise<void>;
  /** Receive decoded lulu packets */
  onMessage(cb: (packet: Packet) => void): void;
  /** Handle connection close */
  onClose(cb: (error?: Error) => void): void;
  /** Handle connection errors */
  onError(cb: (error: Error) => void): void;
  /** Close the connection */
  close(): void;
}

/**
 * Streaming buffer that accumulates raw bytes and emits complete lulu packets.
 */
class PacketStream {
  private buffer = new Uint8Array(0);
  private onPacket: ((p: Packet) => void) | null = null;
  private onError: ((e: Error) => void) | null = null;

  setHandlers(
    onPacket: (p: Packet) => void,
    onError: (e: Error) => void
  ): void {
    this.onPacket = onPacket;
    this.onError = onError;
  }

  feed(data: Uint8Array): void {
    // Append new data
    const combined = new Uint8Array(this.buffer.length + data.length);
    combined.set(this.buffer, 0);
    combined.set(data, this.buffer.length);
    this.buffer = combined;

    // Parse all complete packets
    while (this.buffer.length > 0) {
      try {
        const result = decodePacket(this.buffer);
        if (!result) break; // incomplete, wait for more data

        this.onPacket?.(result.packet);
        this.buffer = this.buffer.slice(result.consumed);
      } catch (err) {
        this.onError?.(
          err instanceof Error ? err : new Error(String(err))
        );
        this.buffer = new Uint8Array(0);
        return;
      }
    }

    // Safety valve: prevent unbounded buffer growth
    if (this.buffer.length > MAX_PACKET_SIZE * 2) {
      this.buffer = new Uint8Array(0);
      this.onError?.(new Error("packet stream buffer overflow"));
    }
  }
}

/**
 * WebSocket connection.
 * Works in both browser and Node.js environments.
 */
export class WebSocketConnection implements IConnection {
  private ws: WebSocket | null = null;
  private closed = false;
  private stream = new PacketStream();
  private connectPromise: Promise<void>;

  constructor(config: ClientConfig) {
    const path = config.wsUpgradePath || "/ws";
    const scheme = config.tls ? "wss" : "ws";
    const url = `${scheme}://${config.addr}${path}`;

    this.ws = new WebSocket(url);
    this.ws.binaryType = "arraybuffer";

    this.connectPromise = new Promise((resolve, reject) => {
      if (!this.ws) return;

      this.ws.onopen = () => resolve();

      this.ws.onerror = () => {
        reject(new Error(`WebSocket connection failed to ${url}`));
      };

      this.ws.onmessage = (event: MessageEvent) => {
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

  async ready(): Promise<void> {
    return this.connectPromise;
  }

  async send(data: Uint8Array): Promise<void> {
    if (!this.ws || this.closed) {
      throw new Error("connection closed");
    }
    this.ws.send(data.buffer);
  }

  onMessage(cb: (packet: Packet) => void): void {
    this.stream.setHandlers(
      cb,
      this._errorCb ? (e) => this._errorCb!(e) : () => {}
    );
  }

  private _errorCb: ((error: Error) => void) | null = null;

  onClose(cb: (error?: Error) => void): void {
    if (!this.ws) return;
    this.ws.onclose = (event: Event) => {
      this.closed = true;
      const wsEvent = event as unknown as { code?: number; reason?: string };
      const err =
        wsEvent.code !== undefined && wsEvent.code !== 1000
          ? new Error(
              `WebSocket closed: code=${wsEvent.code} reason=${wsEvent.reason}`
            )
          : undefined;
      cb(err);
    };
  }

  onError(cb: (error: Error) => void): void {
    this._errorCb = cb;
    this.stream.setHandlers(
      this.stream["onPacket"] ? (p) => this.stream["onPacket"]?.(p) : () => {},
      cb
    );
    if (this.ws) {
      this.ws.onerror = () => cb(new Error("WebSocket error"));
    }
  }

  close(): void {
    if (this.ws && !this.closed) {
      this.closed = true;
      this.ws.close(1000);
    }
  }
}

/**
 * TCP connection.
 * Only available in Node.js environments (uses the `net` module).
 */
export class TcpConnection implements IConnection {
  private socket: import("net").Socket | null = null;
  private closed = false;
  private stream = new PacketStream();
  private connectPromise: Promise<void>;
  private _errorCb: ((error: Error) => void) | null = null;

  constructor(config: ClientConfig) {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const net: typeof import("net") = require("net");

    const [host, portStr] = config.addr.split(":");
    const port = parseInt(portStr, 10);

    if (isNaN(port)) {
      throw new Error(
        `invalid address: ${config.addr} (expected "host:port")`
      );
    }

    this.connectPromise = new Promise((resolve, reject) => {
      const socket = new net.Socket();
      this.socket = socket;

      socket.connect(port, host, () => resolve());

      socket.on("data", (chunk: Buffer) => {
        this.stream.feed(new Uint8Array(chunk));
      });

      socket.on("close", () => {
        this.closed = true;
      });

      socket.on("error", (err: Error) => {
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

  async ready(): Promise<void> {
    return this.connectPromise;
  }

  async send(data: Uint8Array): Promise<void> {
    if (!this.socket || this.closed) {
      throw new Error("connection closed");
    }
    return new Promise((resolve, reject) => {
      this.socket!.write(Buffer.from(data), (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  onMessage(cb: (packet: Packet) => void): void {
    this.stream.setHandlers(
      cb,
      this._errorCb ? (e) => this._errorCb!(e) : () => {}
    );
  }

  onClose(cb: (error?: Error) => void): void {
    if (!this.socket) return;
    this.socket.on("close", () => {
      this.closed = true;
      cb();
    });
  }

  onError(cb: (error: Error) => void): void {
    this._errorCb = cb;
  }

  close(): void {
    if (this.socket && !this.closed) {
      this.closed = true;
      this.socket.destroy();
    }
  }
}
