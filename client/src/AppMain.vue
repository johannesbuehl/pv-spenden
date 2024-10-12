<script setup lang="ts">
	import { ref } from 'vue';

	import BasePV from './components/BasePV.vue';
	import { reserved_modules } from './Globals';

	const selected_module = ref<string>("");

	function get_tooltip_text(id: string): string {
		if (reserved_modules.value[id]) {
			return reserved_modules.value[id];
		} else {
			return "Anonym";
		}
	}
</script>

<template>
	<BasePV
		v-model:selected_module="selected_module"
	>
		<div
			v-if="reserved_modules[selected_module] !== undefined"
			id="tooltip-sold"
		>
			Gespendet von<br>
			{{ get_tooltip_text(selected_module) }}
		</div>
		<div
			v-else
			id="tooltip-buy"
			>
			Dieses Modul spenden<br>
			<a href="https://www.evkirchebuehl.de">Dummy-Link</a>
		</div>
	</BasePV>
</template>

<style scoped>
	
</style>
