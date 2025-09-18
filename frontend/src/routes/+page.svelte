<script lang="ts">
	import { onMount } from 'svelte';

	let activeTab = 'stats';
	let stats = { accepted: 0, rejected: 0, uptime: '', reconnects: 0 };
	let logs: string[] = [];
	let devices: { id: number; lastEventTime: string; lastEvent: string }[] = [];
	let sortField = 'id';
	let sortDirection = 'asc';

	async function updateStats() {
		try {
			stats = await window.go.main.App.GetStats();
		} catch (error) {
			console.error('Помилка при отриманні статистики:', error);
		}
	}

	async function updateLogs() {
		try {
			logs = await window.go.main.App.GetLogs();
		} catch (error) {
			console.error('Помилка при отриманні логів:', error);
		}
	}

	async function updateDevices() {
		try {
			const devicesData = await window.go.main.App.GetDevices();
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

	onMount(() => {
		updateStats();
		updateLogs();
		updateDevices();
		const interval = setInterval(() => {
			updateStats();
			updateLogs();
			updateDevices();
		}, 1000);
		return () => clearInterval(interval);
	});
</script>

<div class="container mx-auto p-2 sm:p-4 max-w-full h-screen flex flex-col">
	<!-- Вкладки -->
	<div class="flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2 mb-4 sm:mb-6">
		<button
			on:click={() => (activeTab = 'stats')}
			class="flex-1 px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl shadow text-center font-semibold transition hover:bg-blue-100"
			class:bg-blue-600={activeTab === 'stats'}
			class:text-white={activeTab === 'stats'}
		>
			📊 Статистика
		</button>
		<button
			on:click={() => (activeTab = 'devices')}
			class="flex-1 px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl shadow text-center font-semibold transition hover:bg-blue-100"
			class:bg-blue-600={activeTab === 'devices'}
			class:text-white={activeTab === 'devices'}
		>
			📋 ППК
		</button>
		<button
			on:click={() => (activeTab = 'logs')}
			class="flex-1 px-2 sm:px-4 py-2 sm:py-3 rounded-lg sm:rounded-xl shadow text-center font-semibold transition hover:bg-blue-100"
			class:bg-blue-600={activeTab === 'logs'}
			class:text-white={activeTab === 'logs'}
		>
			📜 Логи
		</button>
	</div>

	<!-- Контент -->
	<div class="flex-1 min-h-0"> <!-- Важливо: min-h-0 дозволяє елементу стискатися -->
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
					<p class="mt-1 text-sm sm:text-base">Аптайм</p>
				</div>
				<div class="bg-orange-500 text-white shadow rounded-lg sm:rounded-xl p-4 sm:p-6 text-center flex flex-col justify-center">
					<p class="text-2xl sm:text-3xl font-bold">{stats.reconnects}</p>
					<p class="mt-1 text-sm sm:text-base">Реконекти</p>
				</div>
			</div>
		{/if}

		{#if activeTab === 'logs'}
			<div class="bg-black text-green-400 font-mono shadow rounded-lg sm:rounded-xl p-2 sm:p-4 h-full overflow-hidden flex flex-col">
				<div class="overflow-y-auto flex-1">
					<ul class="space-y-1">
						{#each logs as log}
							<li class="whitespace-pre-wrap break-words">{log}</li>
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
								<th class="px-2 sm:px-4 py-2 text-left cursor-pointer" on:click={() => handleSort('id')}>
									<div class="flex items-center">
										<span>№ ППК</span>
										{#if sortField === 'id'}
											<span class="ml-1">{sortDirection === 'asc' ? '↑' : '↓'}</span>
										{/if}
									</div>
								</th>
								<th class="px-2 sm:px-4 py-2 text-left cursor-pointer" on:click={() => handleSort('lastEventTime')}>
									<div class="flex items-center">
										<span>Час останньої події</span>
										{#if sortField === 'lastEventTime'}
											<span class="ml-1">{sortDirection === 'asc' ? '↑' : '↓'}</span>
										{/if}
									</div>
								</th>
								<th class="px-2 sm:px-4 py-2 text-left cursor-pointer" on:click={() => handleSort('lastEvent')}>
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
								<tr class:bg-gray-50={i % 2 === 0} class="hover:bg-blue-50">
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