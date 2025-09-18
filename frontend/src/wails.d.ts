interface Wails {
	main: {
		App: {
			GetStats: () => Promise<{
				accepted: number;
				rejected: number;
				uptime: string;
				reconnects: number;
			}>;
			GetLogs: () => Promise<string[]>;
			GetDevices: () => Promise<
				{ id: number; lastEventTime: string; lastEvent: string }[]
			>;
		};
	};
}

declare global {
	interface Window {
		go: Wails;
	}
}

export {};