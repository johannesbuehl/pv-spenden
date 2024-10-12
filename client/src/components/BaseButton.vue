<script setup lang="ts">

	defineProps<{
		disabled?: boolean;
	}>();
</script>

<template>
	<div class="base-button" :class="{ disabled }" @click="(e) => disabled ? e.stopImmediatePropagation() : undefined">
		<slot></slot>
	</div>
</template>

<style scoped>
	.base-button {
		cursor: pointer;
	}

	.base-button:hover {
		filter: opacity(0.8);
	}

	.base-button.disabled {
		filter: opacity(0.5);

		cursor: not-allowed;
	}
</style>
