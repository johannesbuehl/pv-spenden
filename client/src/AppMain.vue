<script setup lang="ts">
	import { ref } from 'vue';

	import BasePV, { get_element_roof, get_element_type, type Element } from './components/BasePV.vue';
	import { reserved_elements } from './Globals';
import AppLayout from './components/AppLayout/AppLayout.vue';

	const selected_element = ref<Element>();
</script>

<template>
	<AppLayout>
		<BasePV
			v-model:selected_element="selected_element"
		>
			<template #header
				v-if="selected_element !== undefined"
			>
				{{ get_element_roof(selected_element?.mid) }}
			</template>
			<template
				v-if="selected_element !== undefined"
			>
				<div
					v-if="reserved_elements[selected_element.mid] !== undefined"
					id="tooltip-sold"
				>
					Gespendet von {{ selected_element.name }}
				</div>
				<div
					v-else
					id="tooltip-buy"
				>
					Dieses {{ get_element_type(selected_element.mid) }} spenden<br>
					<a href="https://www.evkirchebuehl.de" target="_blank" rel="noopener noreferrer">Dummy-Link</a>
				</div>
			</template>
		</BasePV>
	</AppLayout>
</template>

<style scoped>
	a {
		text-decoration: underline;
		font-style: italic;
	}
</style>
