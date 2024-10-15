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
		position: relative;

		display: flex;

		align-items: baseline;
		justify-content: center;

		width: 100%;
	}

	.bar > * {
		display: flex;

		align-items: baseline;

		gap: 2em;
	}

	#left {
		position: absolute;
		left: 0;
	}

	#right {
		position: absolute;
		right: 0;
	}
</style>
