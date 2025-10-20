import { useEffect, useRef, useState } from "react";

export interface Message {
	type: string;
	data: unknown;
}

export interface DirectoryInfo {
	name: string;
	path: string;
}

export interface DirectoryListData {
	directories: DirectoryInfo[];
}

export interface Agent {
	name: string;
	description?: string;
	mode: string;
	builtIn: boolean;
}

export interface ServerInfo {
	url: string;
	directory: string;
	shareUrl: string;
	sessionId: string;
	currentModel: string;
	currentAgent: string;
	models: string[];
	agents: Agent[];
}

export function useWebSocket() {
	const [connected, setConnected] = useState(false);
	const [directories, setDirectories] = useState<DirectoryInfo[]>([]);
	const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null);
	const wsRef = useRef<WebSocket | null>(null);

	useEffect(() => {
		let isMounted = true;
		const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
		const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
		wsRef.current = ws;

		ws.onopen = () => {
			if (isMounted) {
				setConnected(true);
			} else {
				ws.close();
			}
		};

		ws.onmessage = (event) => {
			if (!isMounted) return;

			const msg: Message = JSON.parse(event.data);

			if (msg.type === "directoryList") {
				const data = msg.data as DirectoryListData;
				setDirectories(data.directories);
			} else if (msg.type === "serverReady") {
				const data = msg.data as ServerInfo;
				setServerInfo(data);
			}
		};

		ws.onclose = () => {
			if (isMounted) {
				setConnected(false);
			}
		};

		ws.onerror = () => {
			// Suppress errors during StrictMode cleanup
		};

		return () => {
			isMounted = false;
			if (ws.readyState === WebSocket.OPEN) {
				ws.close();
			}
		};
	}, []);

	const selectDirectory = (path: string) => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			wsRef.current.send(
				JSON.stringify({
					type: "selectDirectory",
					data: { path },
				}),
			);
		}
	};

	return {
		connected,
		directories,
		serverInfo,
		selectDirectory,
	};
}
