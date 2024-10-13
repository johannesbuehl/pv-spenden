<script setup lang="ts">
	import { inject } from "vue";

	import AppLayout from "@/components/AppLayout/AppLayout.vue";
	import MarkDown from "./components/MarkDown.vue";

	const content_url = inject("content");
</script>

<template>
	<AppLayout>
		<MarkDown v-if="typeof content_url === 'string'" id="markdown" :url="content_url" />
	</AppLayout>
</template>

<style scoped>
	#markdown {
		max-width: 50em;
	}

	#markdown:deep(h1:first-of-type) {
		text-align: center;
	}
</style>
