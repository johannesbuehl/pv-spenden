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
				return `${get_element_type(mid)} ${mid.slice(3).toUpperCase()} (${roof})`;
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
		// svg_dom.removeAttribute("viewBox")
		svg_dom.id = "main-content";

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
	<div
		id="wrapper"
	>
		<div
			v-if="!!svg"
			id="div-svg"
			ref="svg_wrapper"
			v-html="prepare_svg(svg, reserved_elements)"
		></div>
	</div>
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
</template>

<style scoped>
	#wrapper {
		position: relative;

		height: 100%;
		width: 100cqw;

		display: flex;

		flex-direction: column;
		justify-content: center;
		overflow-x: auto;
	}

	#div-svg {
		margin-inline: auto;
	}

	#tooltip-wrapper {
		position: absolute;

		inset: 0;

		backdrop-filter: blur(0.125em);

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
	svg#main-content {
		width: 50em;
		/* max-width: 90cqw; */
		max-width: 100cqh;
	}

	/* @media screen and (max-width: 900px) {
		svg#main-content {
			max-width: none;
		}
	} */

	svg#main-content * {
		user-select: none;
	}

	svg#main-content .element {
		cursor: pointer;
		
		transition: fill 0.2s;
	}

	/* module */
	svg#main-content .module .fill,
	svg#main-content .module.fill {
		fill: var(--color-module);
	}

	svg#main-content .module:hover .fill,
	svg#main-content .module:hover.fill {
		fill: var(--color-module-hover);
	}

	svg#main-content .module.selected .fill,
	svg#main-content .module.selected.fill {
		fill: var(--color-module-selected);
	}

	/* element - sold */
	svg#main-content .module.sold .fill,
	svg#main-content .module.sold.fill {
		fill: var(--color-module-sold);
	}

	svg#main-content .module.sold:hover .fill,
	svg#main-content .module.sold:hover.fill {
		fill: var(--color-module-sold-hover);
	}

	svg#main-content .module.sold.selected .fill,
	svg#main-content .module.sold.selected.fill {
		fill: var(--color-module-sold-selected);
	}

	/* inverter */
	svg#main-content .inverter .fill,
	svg#main-content .inverter.fill {
		fill: var(--color-inverter);
	}

	svg#main-content .inverter:hover .fill,
	svg#main-content .inverter:hover.fill {
		fill: var(--color-inverter-hover);
	}

	svg#main-content .inverter.selected .fill,
	svg#main-content .inverter.selected.fill {
		fill: var(--color-inverter-selected);
	}

	/* inverter - sold */
	svg#main-content .inverter.sold .fill,
	svg#main-content .inverter.sold.fill {
		fill: var(--color-inverter-sold);
	}

	svg#main-content .inverter.sold:hover .fill,
	svg#main-content .inverter.sold:hover.fill {
		fill: var(--color-inverter-sold-hover);
	}

	svg#main-content .inverter.sold.selected .fill,
	svg#main-content .inverter.sold.selected.fill {
		fill: var(--color-inverter-sold-selected);
	}

	/* battery */	
	svg#main-content .battery .fill,
	svg#main-content .battery.fill {
		fill: var(--color-battery);
	}

	svg#main-content .battery:hover .fill,
	svg#main-content .battery:hover.fill {
		fill: var(--color-battery-hover);
	}

	svg#main-content .battery.selected .fill,
	svg#main-content .battery.selected.fill {
		fill: var(--color-battery-selected);
	}

	/* battery - sold */
	svg#main-content .battery.sold .fill,
	svg#main-content .battery.sold.fill {
		fill: var(--color-battery-sold);
	}
	
	svg#main-content .battery.sold:hover .fill,
	svg#main-content .battery.sold:hover.fill {
		fill: var(--color-battery-sold-hover);
	}

	svg#main-content .battery.sold.selected .fill,
	svg#main-content .battery.sold.selected.fill {
		fill: var(--color-battery-soldselected);
	}
</style>