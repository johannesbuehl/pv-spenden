<script lang="ts">
	export interface Module { mid: string; name?: string; }

	export function prepare_svg(r: string, reserved_modules: ReservedModules): string {
		const parser = new DOMParser();
		const svg_dom = parser.parseFromString(r, "image/svg+xml").documentElement;

		svg_dom.removeAttribute("width");
		svg_dom.removeAttribute("height");

		// const pv_module_rects: SVGRectElement[] = Array.from(svg_dom.querySelectorAll(".pv-module"));
		const pv_module_rects: SVGRectElement[] = Array.from(svg_dom.querySelectorAll("path[id^='pv-']"));

		pv_module_rects.forEach(pv_module => {
			pv_module.style.removeProperty("fill");
			pv_module.classList.add("pv-module")

			if (reserved_modules[pv_module.id] !== undefined) {
				pv_module.classList.add("sold");
			}
		});

		let svg_string = new XMLSerializer().serializeToString(svg_dom);

		return svg_string;
	}
</script>

<script setup lang="ts">
	import { onBeforeMount, onMounted, onUnmounted, ref } from 'vue';
	
	import { reserved_modules, type ReservedModules } from '@/Globals';

	import BaseTooltip from './BaseTooltip.vue';

	const svg = ref<string>();

	const svg_path = "modules.svg";

	const svg_wrapper = ref<HTMLDivElement>();
	const tooltip = ref<HTMLDivElement>();
	const selected_module_rect = ref<SVGRectElement>();

	const selected_module = defineModel<Module | undefined>("selected_module");

	onBeforeMount(async () => {
		const svg_request = fetch(svg_path);

		if ((await svg_request).ok) {
			svg.value = await (await svg_request).text()
		}
	});

	function hide_tooltip() {
		if (selected_module_rect.value) {
			selected_module_rect.value.classList.remove("selected");
		}

		selected_module_rect.value = undefined;
		selected_module.value = undefined;
	}

	function on_click(e: MouseEvent) {
		const target = e.target as SVGElement;

		hide_tooltip();

		if (target.classList.contains("pv-module")) {
			// only select the element, if it isn't the previous selected element
			if (target.id !== selected_module_rect.value?.id) {
				selected_module_rect.value = target as SVGRectElement;
				const mid = selected_module_rect.value?.id;

				let reserved_module_text = reserved_modules.value[mid];

				selected_module.value = {
					mid,
					name: reserved_module_text !== "" ? reserved_module_text : "Anonym"
				};
				
				selected_module_rect.value.classList.add("selected");
			}
		}
	}

	onMounted(() => {
		document.addEventListener("click", on_click);
	});

	onUnmounted(() => {
		document.removeEventListener("click", on_click);
	});
</script>

<template>
	<div id="wrapper">
		<div
			v-if="!!svg"
			id="div-svg"
			ref="svg_wrapper"
			v-html="prepare_svg(svg, reserved_modules)"
		></div>
		<Transition>
			<div
				v-if="selected_module"
				id="tooltip-wrapper"
				ref="tooltip"
			>
				<BaseTooltip
					@close="hide_tooltip"
				>
					<template #header>
						<slot name="header"></slot>
					</template>
					<slot></slot>
				</BaseTooltip>
			</div>
		</Transition>
	</div>
</template>

<style scoped>
	#wrapper {
		position: relative;

		align-items: center;

		overflow: auto;
	}

	#div-svg {		
		min-width: 50em;
	}

	#tooltip-wrapper {
		position: absolute;

		inset: 0;

		backdrop-filter: blur(10px);

		display: flex;

		align-items: center;
		justify-content: center;
	}

	.v-enter-active,
	.v-leave-active {
		transition: filter 0.2s;
	}

	.v-enter-from,
	.v-leave-to {
		filter: opacity(0);

	}
</style>

<style>
	svg * {
		user-select: none;
	}

	svg .pv-module:hover {
		fill: hsl(from blue h s 60%);
	}

	svg .pv-module {
		cursor: pointer;
		fill: hsl(240 100% 50%);
		
		transition: fill 0.2s;
	}

	svg .pv-module.selected {
		fill: hsl(210 100% 50%);
	}

	svg .pv-module.sold {
		fill: hsl(240 20% 55%)
	}
</style>
