/**
 * Network transport types supported by lulu.
 */
export type NetworkType = "tcp" | "websocket";
/**
 * Configuration for a lulu client connection.
 */
export interface ClientConfig {
    /** Server address in "host:port" format, e.g. "127.0.0.1:8007" */
    addr: string;
    /** Transport protocol (tcp or websocket) */
    network: NetworkType;
    /** WebSocket upgrade path (only used when network is "websocket"), default "/ws" */
    wsUpgradePath?: string;
    /** Whether to use TLS (wss:// for WebSocket, TLS socket for TCP) */
    tls?: boolean;
    /** Read timeout in seconds, 0 means no timeout */
    readTimeout?: number;
    /** Write timeout in seconds, 0 means no timeout */
    writeTimeout?: number;
    /** Custom heartbeat opcode — if set, sends a ping packet with this opcode */
    heartbeatOpcode?: number;
    /** Heartbeat interval in seconds, default 30 */
    heartbeatInterval?: number;
}
/**
 * A decoded lulu protocol packet.
 */
export interface Packet {
    /** Message opcode */
    opcode: number;
    /** Message body as raw bytes */
    body: Uint8Array;
}
/**
 * Events emitted by the Client.
 */
export interface ClientEvents {
    /** A complete packet was received */
    message: (packet: Packet) => void;
    /** The connection was closed */
    close: (error?: Error) => void;
    /** A fatal error occurred */
    error: (error: Error) => void;
    /** The connection was established */
    connect: () => void;
}
