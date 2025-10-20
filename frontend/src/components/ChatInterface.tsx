import { useRef, useState } from "react";
import { createOpencodeClient } from "@opencode-ai/sdk";
import type { ServerInfo } from "../hooks/useWebSocket";

interface Props {
	serverInfo: ServerInfo;
}

export function ChatInterface({ serverInfo }: Props) {
	const [input, setInput] = useState("");
	const [currentModel, setCurrentModel] = useState(serverInfo.currentModel);
	const [currentAgent, setCurrentAgent] = useState(serverInfo.currentAgent);
	const [showModelPicker, setShowModelPicker] = useState(false);
	const [showAgentPicker, setShowAgentPicker] = useState(false);
	const [isSending, setIsSending] = useState(false);
	const iframeRef = useRef<HTMLIFrameElement>(null);
	const clientRef = useRef(
		createOpencodeClient({
			baseUrl: `/api/${serverInfo.sessionId}`,
		}),
	);

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!input.trim() || isSending) return;

		const userMessage = input;
		setInput("");
		setIsSending(true);

		try {
			const client = clientRef.current;
			await client.session.prompt({
				path: { id: serverInfo.sessionId },
				body: {
					parts: [
						{
							type: "text",
							text: userMessage,
						},
					],
				},
			});
		} catch (error) {
			console.error("Error sending prompt:", error);
		} finally {
			setIsSending(false);
		}
	};

	const handleModelChange = async (model: string) => {
		setCurrentModel(model);
		setShowModelPicker(false);

		try {
			const client = clientRef.current;
			await client.config.update({
				body: { model },
			});
		} catch (error) {
			console.error("Error changing model:", error);
		}
	};

	const handleAgentChange = async (agent: string) => {
		setCurrentAgent(agent);
		setShowAgentPicker(false);

		try {
			const client = clientRef.current;
			const agentConfig: Record<string, { disable: boolean }> = {};

			for (const a of serverInfo.agents) {
				agentConfig[a.name] = {
					disable: a.name !== agent,
				};
			}

			await client.config.update({
				body: {
					agent: agentConfig,
				},
			});
		} catch (error) {
			console.error("Error changing agent:", error);
		}
	};

	return (
		<div className="flex flex-col h-screen bg-zinc-950">
			<div className="p-4 border-b border-zinc-800">
				<div className="flex items-center justify-between mb-2">
					<h2 className="text-lg font-semibold">OpenCode</h2>
					<div className="flex gap-2">
						<button
							onClick={() => {
								setShowAgentPicker(!showAgentPicker);
								setShowModelPicker(false);
							}}
							className="text-sm px-3 py-1 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
						>
							{currentAgent || "Agent"}
						</button>
						<button
							onClick={() => {
								setShowModelPicker(!showModelPicker);
								setShowAgentPicker(false);
							}}
							className="text-sm px-3 py-1 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
						>
							{currentModel || "Model"}
						</button>
					</div>
				</div>
				<div className="text-xs text-zinc-500 truncate">
					{serverInfo.directory}
				</div>

				{showAgentPicker && (
					<div className="mt-3 max-h-64 overflow-y-auto bg-zinc-900 rounded-lg border border-zinc-700">
						{serverInfo.agents.map((agent) => (
							<button
								key={agent.name}
								onClick={() => handleAgentChange(agent.name)}
								className={`w-full text-left px-3 py-2 hover:bg-zinc-800 transition-colors ${
									agent.name === currentAgent ? "bg-zinc-800" : ""
								}`}
							>
								<div
									className={`text-sm ${agent.name === currentAgent ? "font-semibold" : ""}`}
								>
									{agent.name}
								</div>
								{agent.description && (
									<div className="text-xs text-zinc-500 mt-0.5">
										{agent.description}
									</div>
								)}
							</button>
						))}
					</div>
				)}

				{showModelPicker && (
					<div className="mt-3 max-h-64 overflow-y-auto bg-zinc-900 rounded-lg border border-zinc-700">
						{serverInfo.models.map((model) => (
							<button
								key={model}
								onClick={() => handleModelChange(model)}
								className={`w-full text-left px-3 py-2 text-sm hover:bg-zinc-800 transition-colors ${
									model === currentModel ? "bg-zinc-800 font-semibold" : ""
								}`}
							>
								{model}
							</button>
						))}
					</div>
				)}
			</div>

			<div className="flex-1 overflow-hidden">
				<iframe
					ref={iframeRef}
					src={serverInfo.shareUrl}
					className="w-full h-full border-0"
					title="OpenCode Session"
				/>
			</div>

			<form onSubmit={handleSubmit} className="p-4 border-t border-zinc-800">
				<div className="flex gap-2 items-end">
					<textarea
						value={input}
						onChange={(e) => setInput(e.target.value)}
						placeholder="Enter your prompt..."
						disabled={isSending}
						rows={3}
						className="flex-1 px-3 py-2 bg-zinc-900 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm disabled:opacity-50 resize-none"
					/>
					<button
						type="submit"
						disabled={isSending}
						className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg font-medium transition-colors active:bg-blue-800 text-sm disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{isSending ? "..." : "Send"}
					</button>
				</div>
			</form>
		</div>
	);
}
