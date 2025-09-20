
<script lang="ts">
	import { onMount } from 'svelte';
	import { getColorByEvent } from '../eventCodes';
	import eventData from '../data/events.json';
	import * as runtime from '$lib/wailsjs/runtime/runtime.js';
	import { GetStats, GetGlobalEvents, GetDeviceEvents, GetDevices} from '$lib/wailsjs/go/main/App';


	 type Stats = {
        accepted: number;
        rejected: number;
        uptime: string;
        reconnects: number;
    };


	let activeTab = $state('stats');
	let stats = $state<Stats>({ accepted: 0, rejected: 0, uptime: "0s", reconnects: 0 });

	let events = $state<{ time: string; device?: number; data: string }[]>([]);
	let devices = $state<{ id: number; lastEventTime: string; lastEvent: string }[]>([]);
	let selectedDevice = $state<number | null>(null);
	let sortField = $state('id');
	let sortDirection = $state('asc');
	let showPeriodicTests = $state(true);
	 // реактивна змінна (оновлюється автоматично)
    // derived-значення (аналог $: в runes mode)
	const filteredEvents = $derived(
		events.filter(ev => {
			if (!showPeriodicTests) {
				const code = ev.data.match(/([ER]\d{3})/)?.[1] ?? '';
				if (code === 'E602') return false;
			}
			return true;
		}).slice(0, 1000) // Додатковий ліміт для рендерингу
	);


	// Create a map for quick lookup of event descriptions
	const eventDescriptionMap: Record<string, { 
		type: string; 
		description: string; 
		zone: number; 
		group: number | null; 
		}[]> = {};
	eventData.forEach(e => {
		if (!eventDescriptionMap[e.contactId_code]) {
			eventDescriptionMap[e.contactId_code] = [];
		}
		eventDescriptionMap[e.contactId_code].push({
			type: e.TypeCodeMes_UK,
			description: e.CodeMes_UK,
			zone: e.Zoneno,
			group: e.GroupSent,
		});
		});

	async function updateStats() {
		try {
			stats = await GetStats();
		} catch (error) {
			console.error('Помилка при отриманні статистики:', error);
		}
	}

	async function updateEvents() {
		try {
			if (selectedDevice === null) {
				const ge = await GetGlobalEvents();
				events = ge.map((e: { time: string; deviceID: number; data: string }) => ({
					time: e.time,
					device: e.deviceID,
					data: e.data
				}));
			} else {
				const de = await GetDeviceEvents(selectedDevice);
				events = de.map((e: { time: string; data: string }) => ({
					time: e.time,
					data: e.data
				}));
			}
		} catch (error) {
			console.error('Помилка при отриманні подій:', error);
		}
	}

	async function updateDevices() {
		try {
			const devicesData = await GetDevices();
			devices = sortDevices(devicesData, sortField, sortDirection);
		} catch (error) {
			console.error('Помилка при отриманні пристроїв:', error);
		}
	}

	function sortDevices(
		devicesData: { id: number; lastEventTime: string; lastEvent: string }[],
		field: string,
		direction: string
	) {
		return [...devicesData].sort((a, b) => {
			let comparison = 0;
			if (field === 'id') {
				comparison = a.id - b.id;
			} else if (field === 'lastEventTime') {
				comparison = new Date(a.lastEventTime).getTime() - new Date(b.lastEventTime).getTime();
			} else if (field === 'lastEvent') {
				comparison = a.lastEvent.localeCompare(b.lastEvent);
			}
			return direction === 'asc' ? comparison : -comparison;
		});
	}

	function handleSort(field: string) {
		if (sortField === field) {
			sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
		} else {
			sortField = field;
			sortDirection = 'asc';
		}
		devices = sortDevices(devices, sortField, sortDirection);
	}

	function handleDeviceClick(id: number) {
		selectedDevice = id;
		activeTab = 'events';
	}

	function goBackToGeneral() {
		selectedDevice = null;
	}

	
    // подальші оновлення приходять по події "stats_update"
    $effect(() => {
        const off = runtime.EventsOn("stats_update", (data: Stats) => {
            stats = data;
        });
        return () => off();
    });

	$effect(() => {
        runtime.EventsOn("logs_update", updateEvents);
    });

	$effect(() => {
        runtime.EventsOn("device_update", updateDevices);
    });
	
	onMount(() => {
    // перший кадр одразу
    updateStats();
    updateEvents();
    updateDevices();

});

	// onMount(() => {
	// 	updateStats();
	// 	updateEvents();
	// 	updateDevices();

	// 	// const interval = setInterval(() => {
	// 	// 	updateStats();
	// 	// 	updateEvents();
	// 	// 	updateDevices();
	// 	// }, 1000);
	// 	// return () => clearInterval(interval);
	// });
</script>

<div class="container mx-auto p-2 sm:p-4 max-w-full h-screen flex flex-col">
	<!-- Вкладки -->
	<div class="flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2 mb-4 sm:mb-6">
		<button
			onclick={() => (activeTab = 'stats')}
			class="flex-1 px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl shadow text-center font-semibold transition hover:bg-blue-100"
			class:bg-blue-600={activeTab === 'stats'}
			class:text-white={activeTab === 'stats'}
		>
			📊 Статистика
		</button>
		<button
			onclick={() => (activeTab = 'devices')}
			class="flex-1 px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl shadow text-center font-semibold transition hover:bg-blue-100"
			class:bg-blue-600={activeTab === 'devices'}
			class:text-white={activeTab === 'devices'}
		>
			📋 ППК
		</button>
		<button
			onclick={() => { activeTab = 'events'; selectedDevice = null; }}
			class="flex-1 px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl shadow text-center font-semibold transition hover:bg-blue-100"
			class:bg-blue-600={activeTab === 'events'}
			class:text-white={activeTab === 'events'}
		>
			📜 Журнал подій
		</button>
	</div>

	<!-- Контент -->
	<div class="flex-1 min-h-0">
		{#if activeTab === 'stats'}
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-4">
				<div class="bg-green-500 text-white shadow rounded-lg sm:rounded-xl p-4 sm:p-6 text-center flex flex-col justify-center">
					<p class="text-2xl sm:text-3xl font-bold">{stats.accepted}</p>
					<p class="mt-1 text-sm sm:text-base">Прийняті</p>
				</div>
				<div class="bg-red-500 text-white shadow rounded-lg sm:rounded-xl p-4 sm:p-6 text-center flex flex-col justify-center">
					<p class="text-2xl sm:text-3xl font-bold">{stats.rejected}</p>
					<p class="mt-1 text-sm sm:text-base">Не прийняті</p>
				</div>
				<div class="bg-blue-500 text-white shadow rounded-lg sm:rounded-xl p-4 sm:p-6 text-center flex flex-col justify-center">
					<p class="text-2xl sm:text-3xl font-bold">{stats.uptime}</p>
					<p class="mt-1 text-sm sm:text-base">Час роботи</p>
				</div>
				<div class="bg-orange-500 text-white shadow rounded-lg sm:rounded-xl p-4 sm:p-6 text-center flex flex-col justify-center">
					<p class="text-2xl sm:text-3xl font-bold">{stats.reconnects}</p>
					<p class="mt-1 text-sm sm:text-base">Перепідключення</p>
				</div>
			</div>
		{/if}

		{#if activeTab === 'events'}
			<div class="bg-black font-mono shadow rounded-lg sm:rounded-xl p-2 sm:p-4 h-full overflow-hidden flex flex-col">
				<!-- Header -->
				<div class="p-2 bg-blue-900 text-white flex items-center">
					{#if selectedDevice !== null}
						<button
							onclick={goBackToGeneral}
							class="mr-4 px-2 py-1 bg-gray-700 rounded hover:bg-gray-600"
						>
							← Назад до загального
						</button>
						<span>Журнал подій ППК #{selectedDevice}</span>
					{:else}
						<span>Загальний журнал подій ППК</span>
						<div class="ml-auto flex items-center gap-2">
							<label>
								<input type="checkbox" checked={showPeriodicTests} />
								Показати періодичні тести
							</label>
						</div>
					{/if}
				</div>
				<!-- Events list -->
				<div class="overflow-y-auto flex-1">
					<ul class="space-y-1 p-2">
						{#each filteredEvents as ev}
							{@const eventCode = ev.data.match(/([ER]\d{3})/)?.[1] ?? ''}
							{@const colorClass = getColorByEvent(eventCode)}
							{@const eventType = eventDescriptionMap[eventCode]?.[0]?.type ?? 'Опис відсутній'}
							{@const eventDescription = eventDescriptionMap[eventCode]?.[0]?.description ?? 'Опис відсутній'}
							<!-- {@const eventZone = eventDescriptionMap[eventCode]?.[0]?.zone ?? ''}
							{@const eventGroup = eventDescriptionMap[eventCode]?.[0]?.group ?? ''} -->
							{@const groupNum = ev.data.slice(15, 17)}
							{@const zoneNum = ev.data.slice(17, 20)}
							<li class="flex items-start text-xs !text-[inherit]">
								<span class="w-36 font-bold min-w-[6rem] {colorClass}">{ev.time}</span>
								{#if selectedDevice === null && ev.device !== undefined}
									<span class="w-8 ml-2 font-bold min-w-[3rem] {colorClass}">{ev.device}</span>
									<span class="w-8 ml-2 min-w-[3rem] {colorClass}">{eventCode}</span>
									<span class="w-48 ml-2 min-w-[3rem] {colorClass}">{eventType}</span>
									<span class="flex-1 ml-2 whitespace-pre-wrap break-words {colorClass}">{eventDescription}</span>
									<span class="w-40 ml-2 min-w-[3rem] {colorClass}">Група: {groupNum}, Зона: {zoneNum}</span>
								{:else}
									<span class="w-16 ml-2 min-w-[4rem] {colorClass}">{eventCode}</span>
									<span class="flex-1 ml-2 whitespace-pre-wrap break-words {colorClass}">{eventDescription}</span>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}

		{#if activeTab === 'devices'}
			<div class="shadow rounded-lg sm:rounded-xl h-full overflow-hidden flex flex-col">
				<div class="overflow-auto flex-1">
					<table class="w-full border-collapse">
						<thead class="bg-gray-200 sticky top-0">
							<tr>
								<th class="px-2 sm:px-4 py-2 text-left cursor-pointer" onclick={() => handleSort('id')}>
									<div class="flex items-center">
										<span>№ ППК</span>
										{#if sortField === 'id'}
											<span class="ml-1">{sortDirection === 'asc' ? '↑' : '↓'}</span>
										{/if}
									</div>
								</th>
								<th class="px-2 sm:px-4 py-2 text-left cursor-pointer" onclick={() => handleSort('lastEventTime')}>
									<div class="flex items-center">
										<span>Час останньої події</span>
										{#if sortField === 'lastEventTime'}
											<span class="ml-1">{sortDirection === 'asc' ? '↑' : '↓'}</span>
										{/if}
									</div>
								</th>
								<th class="px-2 sm:px-4 py-2 text-left cursor-pointer" onclick={() => handleSort('lastEvent')}>
									<div class="flex items-center">
										<span>Остання подія</span>
										{#if sortField === 'lastEvent'}
											<span class="ml-1">{sortDirection === 'asc' ? '↑' : '↓'}</span>
										{/if}
									</div>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each devices as d, i}
								{@const now = new Date().getTime()}
								{@const lastTime = new Date(d.lastEventTime).getTime()}
								{@const isRecent = (now - lastTime) < 15 * 60 * 1000}
								<tr 
									onclick={() => handleDeviceClick(d.id)}
									class="cursor-pointer hover:bg-blue-50"
									class:bg-red-200={!isRecent}
									class:bg-green-100={isRecent}
									class:bg-gray-50={i % 2 === 0 && isRecent}
								>
									<td class="px-2 sm:px-4 py-2 text-xs sm:text-sm">{d.id}</td>
									<td class="px-2 sm:px-4 py-2 text-xs sm:text-sm">{d.lastEventTime}</td>
									<td class="px-2 sm:px-4 py-2 text-xs sm:text-sm">{d.lastEvent}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.container {
		max-height: 100vh;
	}
	@media (max-width: 640px) {
		th, td {
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
			max-width: 120px;
		}
	}
</style>
