import { useState } from "react";
import { ChatInterface } from "./components/ChatInterface";
import { DirectorySelector } from "./components/DirectorySelector";
import { useWebSocket } from "./hooks/useWebSocket";

function App() {
	const { connected, directories, serverInfo, selectDirectory } =
		useWebSocket();
	const [loading, setLoading] = useState(false);

	const handleSelectDirectory = (path: string) => {
		setLoading(true);
		selectDirectory(path);
	};

	if (!connected) {
		return (
			<div className="flex items-center justify-center h-screen bg-zinc-950 text-zinc-100">
				<div className="text-center">
					<div className="animate-pulse text-lg">Connecting...</div>
				</div>
			</div>
		);
	}

	if (loading && !serverInfo) {
		return (
			<div className="flex items-center justify-center h-screen bg-zinc-950 text-zinc-100">
				<div className="text-center">
					<div className="animate-pulse text-lg mb-2">
						Starting OpenCode server...
					</div>
					<div className="text-sm text-zinc-500">This may take a moment</div>
				</div>
			</div>
		);
	}

	if (!serverInfo) {
		return (
			<DirectorySelector
				directories={directories}
				onSelect={handleSelectDirectory}
			/>
		);
	}

	return <ChatInterface serverInfo={serverInfo} />;
}

export default App;
