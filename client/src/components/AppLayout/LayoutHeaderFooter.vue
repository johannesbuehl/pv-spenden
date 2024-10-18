<script setup lang="ts"></script>

<template>
	<div class="bar">
		<div id="left">
			<slot name="left"></slot>
		</div>
		<div>
			<slot></slot>
		</div>
		<div id="right">
			<slot name="right"></slot>
		</div>
	</div>
</template>

<style scoped>
	.bar {
		display: flex;

		align-items: baseline;
		justify-content: space-between;

		width: 100%;
	}

	.bar > div {
		display: flex;

		align-items: baseline;

		font-family: var(--font-header);

		column-gap: 2em;
	}
</style>
