import { useState } from "react";
import type { DirectoryInfo } from "../hooks/useWebSocket";

interface Props {
	directories: DirectoryInfo[];
	onSelect: (path: string) => void;
}

export function DirectorySelector({ directories, onSelect }: Props) {
	const [filter, setFilter] = useState("");

	const filteredDirs = directories.filter((dir) =>
		dir.name.toLowerCase().includes(filter.toLowerCase()),
	);

	return (
		<div className="flex flex-col h-screen bg-zinc-950">
			<div className="p-4 border-b border-zinc-800">
				<h2 className="text-lg font-semibold mb-3">Select Directory</h2>
				<input
					type="text"
					placeholder="Filter directories..."
					value={filter}
					onChange={(e) => setFilter(e.target.value)}
					className="w-full px-3 py-2 bg-zinc-900 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
				/>
			</div>
			<div className="flex-1 overflow-y-auto">
				{filteredDirs.map((dir) => (
					<button
						key={dir.path}
						onClick={() => onSelect(dir.path)}
						className="w-full px-4 py-3 text-left hover:bg-zinc-900 border-b border-zinc-900 transition-colors active:bg-zinc-800"
					>
						<div className="font-medium">{dir.name}</div>
						<div className="text-xs text-zinc-500 mt-1 truncate">
							{dir.path}
						</div>
					</button>
				))}
			</div>
		</div>
	);
}
