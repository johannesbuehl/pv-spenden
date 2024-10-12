<script setup lang="ts">
import { onMounted, onUpdated } from 'vue';

	const emit = defineEmits<{
		refresh: [];
	}>();

	onMounted(() => {
		emit("refresh");
	});

	onUpdated(() => {
		emit("refresh");
	});
</script>

<template>
	<div
		id="tooltip"
		@click="(e) => { e.stopPropagation()}"
	>
		<slot>
			
		</slot>	
	</div>
</template>

<style scoped>
	#tooltip {
		background-color: white;

		border-radius: 0.125em;
		padding: 0.25em;

		box-shadow: 0.125em 0.125em 0.25em;

		text-align: center;

		transform: translateY(0.125em);
	}
</style>
