<script setup lang="ts">
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import BaseButton from './BaseButton.vue';
import { faXmark } from '@fortawesome/free-solid-svg-icons';

	defineEmits<{close: [] }>();
</script>

<template>
	<div
		id="tooltip"
		@click="(e) => { e.stopPropagation()}"
	>
		<div id="header">
			<slot name="header"></slot>
			<BaseButton id="btn-close" @click="$emit('close')"><FontAwesomeIcon :icon="faXmark" /></BaseButton>
		</div>
		<div id="text">
			<slot></slot>	
		</div>
	</div>
</template>

<style scoped>
	#tooltip {
		background-color: white;

		border-radius: 0.125em;
		padding: 0.25em;

		box-shadow: 0.125em 0.125em 0.25em;

		max-width: fit-content;

		display: flex;
		flex-direction: column;
		gap: 0.5em;
	}

	#header {
		display: flex;

		gap: 1em;

		font-family: Aleo;
		font-weight: bold;
	}

	#btn-close {
		margin-left: auto;
	}
</style>
