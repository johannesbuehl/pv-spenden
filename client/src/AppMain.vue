<script lang="ts">
	const row_roof_map = {
		d: "Kirchendach",
		r: "Gemeindehaus",
		v: "Pfarrhaus"
	}

	export function get_module_roof(mid: string): string {
		const mid_row = mid.slice(3, 4);

		for (let [row, roof] of Object.entries(row_roof_map)) {
			if (mid_row <= row) {
				return `Modul "${mid.slice(3).toUpperCase()}" (${roof})`;
			}
		}

		return mid_row;
	}
</script>

<script setup lang="ts">
	import { ref } from 'vue';

	import BasePV, { type Module } from './components/BasePV.vue';
	import { reserved_modules } from './Globals';
import AppLayout from './components/AppLayout/AppLayout.vue';

	const selected_module = ref<Module>();
</script>

<template>
	<AppLayout>
		<BasePV
		v-model:selected_module="selected_module"
		>
			<template #header
				v-if="selected_module !== undefined"
			>
				{{ get_module_roof(selected_module?.mid) }}
			</template>
			<div
			v-if="selected_module && reserved_modules[selected_module.mid] !== undefined"
			id="tooltip-sold"
			>
			Gespendet von<br>
			{{ selected_module.name }}
		</div>
		<div
			v-else
			id="tooltip-buy"
		>
			Dieses Modul spenden<br>
			<a href="https://www.evkirchebuehl.de" target="_blank" rel="noopener noreferrer">Dummy-Link</a>
		</div>
		</BasePV>
	</AppLayout>
</template>

<style scoped>
	
</style>
