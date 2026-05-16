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
import type { ClientConfig, ClientEvents } from "./types";
export declare class LuluClient {
    private config;
    private conn;
    private listeners;
    private heartbeatTimer;
    private connected;
    constructor(config: ClientConfig);
    /**
     * Connect to the lulu server.
     */
    connect(): Promise<void>;
    /**
     * Send a message to the server.
     *
     * @param opcode - The message opcode (0–65535)
     * @param body - The protobuf-encoded message body
     */
    send(opcode: number, body: Uint8Array): Promise<void>;
    /**
     * Register an event listener.
     */
    on<E extends keyof ClientEvents>(event: E, listener: ClientEvents[E]): this;
    /**
     * Remove an event listener.
     */
    off<E extends keyof ClientEvents>(event: E, listener: ClientEvents[E]): this;
    /**
     * Close the connection.
     */
    close(): void;
    get isConnected(): boolean;
    private createConnection;
    private startHeartbeat;
    private stopHeartbeat;
    private emit;
}
