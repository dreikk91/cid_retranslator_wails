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
			GetGlobalEvents: () => Promise<
				{ time: string; deviceID: number; data: string }[]
			>;
			GetDeviceEvents: (id: number) => Promise<
				{ time: string; data: string }[]
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