/**
 * Connection abstraction for lulu transports.
 *
 * Supports:
 * - WebSocket (browser and Node.js)
 * - TCP (Node.js only, via the `net` module)
 */
import type { ClientConfig, Packet } from "./types";
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
 * WebSocket connection.
 * Works in both browser and Node.js environments.
 */
export declare class WebSocketConnection implements IConnection {
    private ws;
    private closed;
    private stream;
    private connectPromise;
    constructor(config: ClientConfig);
    ready(): Promise<void>;
    send(data: Uint8Array): Promise<void>;
    onMessage(cb: (packet: Packet) => void): void;
    private _errorCb;
    onClose(cb: (error?: Error) => void): void;
    onError(cb: (error: Error) => void): void;
    close(): void;
}
/**
 * TCP connection.
 * Only available in Node.js environments (uses the `net` module).
 */
export declare class TcpConnection implements IConnection {
    private socket;
    private closed;
    private stream;
    private connectPromise;
    private _errorCb;
    constructor(config: ClientConfig);
    ready(): Promise<void>;
    send(data: Uint8Array): Promise<void>;
    onMessage(cb: (packet: Packet) => void): void;
    onClose(cb: (error?: Error) => void): void;
    onError(cb: (error: Error) => void): void;
    close(): void;
}
