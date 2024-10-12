<script setup lang="ts">
import { faCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

	const checked = defineModel<boolean>();
</script>

<template>
	<div
		@click="checked = !checked"
		id="checkbox-wrapper"
		:class="{ checked }"
	>
		<FontAwesomeIcon :icon="faCheck" />
	</div>
</template>

<style scoped>
	#checkbox-wrapper {
		border-radius: 0.125em;

		border: 0.0625em solid gray;

		padding: 0.0625em;

		min-height: 0;
		aspect-ratio: 1;

		display: flex;
		align-items: center;
		justify-content: center;

		cursor: pointer;

		transition: border 0.2s, background-color 0.2s;
	}

	#checkbox-wrapper:hover {
		border-color: hsl(0, 0%, 31%);
	}

	#checkbox-wrapper.checked {
		color: white;
		background-color: hsl(212, 100%, 50%);
		border-color: hsl(212, 100%, 50%);
	}

	#checkbox-wrapper.checked:hover {
		background-color: hsl(212, 100%, 39%);
		border-color: hsl(212, 100%, 39%);
	}

	#checkbox-wrapper:not(.checked) > svg {
		opacity: 0;
	}
</style>
