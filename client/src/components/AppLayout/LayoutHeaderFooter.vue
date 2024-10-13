<script setup lang="ts"></script>

<template>
	<div class="bar">
		<div>
			<slot name="left"></slot>
		</div>
		<div>
			<slot></slot>
		</div>
		<div>
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

	.bar > * {
		display: flex;

		align-items: baseline;

		gap: 2em;
	}
</style>
