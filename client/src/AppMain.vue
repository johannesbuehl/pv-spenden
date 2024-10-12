<script setup lang="ts">
	import { ref } from 'vue';

	import BasePV, { type Module } from './components/BasePV.vue';
	import { reserved_modules } from './Globals';

	const selected_module = ref<Module>();
</script>

<template>
	<BasePV
		v-model:selected_module="selected_module"
	>
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
			<a href="https://www.evkirchebuehl.de">Dummy-Link</a>
		</div>
	</BasePV>
</template>

<style scoped>
	
</style>
