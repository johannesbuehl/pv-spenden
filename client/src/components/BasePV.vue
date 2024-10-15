<script lang="ts">
	export interface Element { mid: string; name?: string; }

	const element_type_map: Record<string, string> = {
			pv: "PV-Modul",
			wr: "Wechselrichter",
			bs: "Batterie"
	}

	export function get_element_type(mid: string): string {
		return element_type_map[mid.slice(0, 2)]
	}

	const row_roof_map = {
		d: "Kirchendach",
		r: "Gemeindehaus",
		v: "Pfarrhaus"
	}

	export function get_element_roof(mid: string): string {
		const mid_row = mid.slice(3, 4);

		for (let [row, roof] of Object.entries(row_roof_map)) {
			if (mid_row <= row) {
				return `${get_element_type(mid)} "${mid.slice(3).toUpperCase()}" (${roof})`;
			}
		}

		return mid_row;
	}
</script>

<script setup lang="ts">
	import { onBeforeMount, onMounted, onUnmounted, ref } from 'vue';
	
	import { reserved_elements, type ReservedElements } from '@/Globals';

	import BaseTooltip from './BaseTooltip.vue';

	const svg = ref<string>();

	const svg_path = "elements.svg";

	const svg_wrapper = ref<HTMLDivElement>();
	const tooltip = ref<HTMLDivElement>();
	const svg_selected_element = ref<SVGRectElement>();

	const selected_element = defineModel<Element | undefined>("selected_element");

	onBeforeMount(async () => {
		const svg_request = fetch(svg_path);

		if ((await svg_request).ok) {
			svg.value = await (await svg_request).text()
		}
	});

	function hide_tooltip() {
		if (svg_selected_element.value) {
			svg_selected_element.value.classList.remove("selected");
		}

		svg_selected_element.value = undefined;
		selected_element.value = undefined;
	}

	function prepare_svg(r: string, reserved_elements: ReservedElements): string {
		const parser = new DOMParser();
		const svg_dom = parser.parseFromString(r, "image/svg+xml").documentElement;

		svg_dom.removeAttribute("width");
		svg_dom.removeAttribute("height");

		const prepare_element = (ele: SVGPathElement, classname: string) => {
			ele.querySelectorAll<SVGSetElement>(".fill").forEach((e) => e.style.removeProperty("fill"));

			ele.style.removeProperty("fill");

			ele.classList.add("element");

			ele.classList.add(classname);

			if (reserved_elements[ele.id] !== undefined) {
				ele.classList.add("sold");
			}
		}

		// select all elements
		const elements: SVGPathElement[] = Array.from(svg_dom.querySelectorAll<SVGPathElement>("[id^='pv-']"));
		elements.forEach(element => prepare_element(element, "module"));

		// select all inverters
		const inverters: SVGPathElement[] = Array.from(svg_dom.querySelectorAll<SVGPathElement>("[id^='wr-']"))
		inverters.forEach(element => prepare_element(element, "inverter"));

		// select all batteries
		const batteries: SVGPathElement[] = Array.from(svg_dom.querySelectorAll<SVGPathElement>("[id^='bs-']"))
		batteries.forEach(element => prepare_element(element, "battery"));

		let svg_string = new XMLSerializer().serializeToString(svg_dom);

		return svg_string;
	}

	onMounted(() => {
		document.addEventListener("click", on_click);
	});

	onUnmounted(() => {
		document.removeEventListener("click", on_click);
	});

	function on_click(e: MouseEvent) {
		const target = (e.target as SVGElement).closest(".element");

		if (target) {
			svg_selected_element.value = target as SVGRectElement;
			const mid = svg_selected_element.value?.id;

			let reserved_element_text = reserved_elements.value[mid];

			selected_element.value = {
				mid,
				name: reserved_element_text !== "" ? reserved_element_text : "Anonym"
			};
			
			svg_selected_element.value.classList.add("selected");
		}
	}
</script>

<template>
	<div id="wrapper">
		<div
			v-if="!!svg"
			id="div-svg"
			ref="svg_wrapper"
			v-html="prepare_svg(svg, reserved_elements)"
		></div>
		<Transition>
			<div
				v-if="selected_element"
				id="tooltip-wrapper"
				ref="tooltip"
				@click="hide_tooltip"
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

		backdrop-filter: blur(0.5em);

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

	svg .element {
		cursor: pointer;
	}

	svg .element .fill,
	svg .element.fill {
		fill: hsl(240 100% 50%);
		
		transition: filter 0.2s;
	}

	svg .element:hover .fill,
	svg .element:hover.fill {
		filter: brightness(75%);
	}

	svg .element.selected .fill,
	svg .element.selected.fill {
		fill: hsl(210 100% 50%);
	}

	svg .element.sold .fill,
	svg .element.sold.fill {
		fill: hsl(240 20% 55%)
	}


	svg .inverter .fill,
	svg .inverter.fill {
		fill: hsl(30 100% 50%);
		
		transition: filter 0.2s;
	}

	svg .inverter:hover .fill,
	svg .inverter:hover.fill {
		filter: brightness(75%);
	}

	svg .inverter.select .fill,
	svg .inverter.select.fill {
		fill: hsl(210 100% 50%);
	}

	svg .inverter.sold .fill,
	svg .inverter.sold.fill {
		fill: hsl(240 20% 55%)
	}

	
	svg .battery .fill,
	svg .battery.fill {
		fill: hsl(120, 50%, 50%);
		
		transition: filter 0.2s;
	}

	svg .battery:hover .fill,
	svg .battery:hover.fill {
		filter: brightness(75%);
	}

	svg .battery.select .fill,
	svg .battery.select.fill {
		fill: hsl(210 100% 50%);
	}

	svg .battery.sold .fill,
	svg .battery.sold.fill {
		fill: hsl(240 20% 55%)
	}
</style>
