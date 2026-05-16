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

import type { ClientConfig, Packet, ClientEvents } from "./types";
import { encodePacket, encodeHeartbeat } from "./packet";
import {
  IConnection,
  WebSocketConnection,
  TcpConnection,
} from "./connection";

type EventListener = (...args: any[]) => void;

export class LuluClient {
  private config: Required<ClientConfig>;
  private conn: IConnection | null = null;
  private listeners: Map<keyof ClientEvents, EventListener[]> = new Map();
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private connected = false;

  constructor(config: ClientConfig) {
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
  async connect(): Promise<void> {
    if (this.connected) {
      throw new Error("already connected");
    }

    const conn = this.createConnection();
    await conn.ready();

    conn.onMessage((packet: Packet) => {
      this.emit("message", packet);
    });

    conn.onClose((error?: Error) => {
      this.connected = false;
      this.stopHeartbeat();
      this.emit("close", error);
    });

    conn.onError((error: Error) => {
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
  async send(opcode: number, body: Uint8Array): Promise<void> {
    if (!this.conn || !this.connected) {
      throw new Error("not connected");
    }
    const raw = encodePacket(opcode, body);
    await this.conn.send(raw);
  }

  /**
   * Register an event listener.
   */
  on<E extends keyof ClientEvents>(
    event: E,
    listener: ClientEvents[E]
  ): this {
    const existing = this.listeners.get(event) || [];
    existing.push(listener as EventListener);
    this.listeners.set(event, existing);
    return this;
  }

  /**
   * Remove an event listener.
   */
  off<E extends keyof ClientEvents>(
    event: E,
    listener: ClientEvents[E]
  ): this {
    const existing = this.listeners.get(event) || [];
    this.listeners.set(
      event,
      existing.filter((l) => l !== listener)
    );
    return this;
  }

  /**
   * Close the connection.
   */
  close(): void {
    this.stopHeartbeat();
    if (this.conn) {
      this.conn.close();
      this.conn = null;
    }
    this.connected = false;
  }

  get isConnected(): boolean {
    return this.connected;
  }

  // ── private ──────────────────────────────────────────────

  private createConnection(): IConnection {
    switch (this.config.network) {
      case "websocket":
        return new WebSocketConnection(this.config);
      case "tcp":
        return new TcpConnection(this.config);
      default:
        throw new Error(
          `unsupported network type: "${this.config.network}"`
        );
    }
  }

  private startHeartbeat(): void {
    if (!this.config.heartbeatOpcode) return;
    const interval = (this.config.heartbeatInterval || 30) * 1000;
    this.heartbeatTimer = setInterval(async () => {
      try {
        const hb = encodeHeartbeat(this.config.heartbeatOpcode);
        await this.conn?.send(hb);
      } catch {
        this.close();
      }
    }, interval);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private emit<E extends keyof ClientEvents>(
    event: E,
    ...args: Parameters<ClientEvents[E]>
  ): void {
    const listeners = this.listeners.get(event) || [];
    for (const listener of listeners) {
      try {
        (listener as (...a: any[]) => void)(...args);
      } catch (err) {
        console.error(`lulu-client: error in "${event}" listener:`, err);
      }
    }
  }
}
