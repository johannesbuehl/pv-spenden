<script lang="ts">
	enum WindowState {
		Login,
		Elements,
		Account,
		Users
	}
</script>

<script setup lang="ts">
	import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
	import { faPlus, faSdCard, faTrashCan, faXmark } from '@fortawesome/free-solid-svg-icons';
	import { ref, watch } from 'vue';
	
	import BasePV, { get_element_roof, get_element_type, type Element } from './components/BasePV.vue';
	import { api_call, type APICallResult } from './lib';
	import { reserved_elements, user, type ReservedElements } from './Globals';
	import AdminUsers from './components/AdminUsers.vue';
	import BaseButton from './components/BaseButton.vue';
	import AdminLogin from './components/AdminLogin.vue';
	import AdminAccount from './components/AdminAccount.vue';
	import AppLayout from './components/AppLayout/AppLayout.vue';

	const window_state = ref<WindowState>(WindowState.Login);
	const selected_element = ref<Element>();

	watch(user, user => {
			window_state.value = user?.logged_in ? WindowState.Elements : WindowState.Login
	}, { deep: true })

	async function submit() {
		// check wether a element is selected
		if (selected_element.value !== undefined) {
			let response: APICallResult<ReservedElements>;
			
			const name = selected_element.value.name !== "" ? selected_element.value.name : "Anonym";
				
			// if the element is already reserved, patch it instead
			const method = reserved_elements.value[selected_element.value.mid] === undefined ? "POST" : "PATCH";
				
			response = await api_call<{ reserved_elements: ReservedElements }>(method, "elements", { mid: selected_element.value.mid }, {
				name
			});
			
			if (response.ok) {
				reserved_elements.value = (await response.json()).reserved_elements;
				
				selected_element.value = undefined;
			} else {
				alert(`Error during database write: ${await response.text()}`);
			}
		}
	}

	async function delete_reservation() {
		// only proceed if there is a valid element-selection
		if (selected_element.value !== undefined) {
			const element_name = selected_element.value?.mid.match(/\w?\d+/)?.["0"].toUpperCase()
			
			if (confirm(`Reservierung für ${get_element_type(selected_element.value.mid)} ${element_name} mit dem Namen "${reserved_elements.value[selected_element.value.mid]}" löschen?`)) {
				const response = await api_call<{ reserved_elements: ReservedElements }>("DELETE", "elements", { mid: selected_element.value.mid });
				
				if (response.ok) {
					reserved_elements.value = (await response.json()).reserved_elements;

					selected_element.value.name = undefined;
				}
			}
		}
	}
</script>

<template>
	<AppLayout>
		<template #header>
			<a class="navbar-item" :class="{ active: window_state === WindowState.Elements }" @click="window_state = WindowState.Elements">Elemente</a>
			<a class="navbar-item" :class="{ active: window_state === WindowState.Account }" @click="window_state = WindowState.Account">Account</a>
			<a v-if="user?.name === 'admin'" class="navbar-item" :class="{ active: window_state === WindowState.Users }" @click="window_state = WindowState.Users">Benutzer</a>
		</template>
		<AdminLogin v-if="window_state === WindowState.Login" v-model="user" />
		<BasePV
			v-else-if="window_state === WindowState.Elements"
			v-model:selected_element="selected_element"
		>
			<template #header
				v-if="selected_element !== undefined"
			>
				{{ get_element_roof(selected_element.mid) }}
			</template>
			<div
				v-if="selected_element"
				id="tooltip_content"
			>
				<BaseButton
					v-if=" selected_element.name === undefined"
					@click="selected_element.name = ''"
				>
					<FontAwesomeIcon :icon="faPlus"></FontAwesomeIcon> {{ get_element_type(selected_element.mid) }} {{ selected_element.mid.match(/\w?\d+/)?.[0].toUpperCase() }} reservieren
				</BaseButton>
				<template v-else>
					<BaseButton @click="submit">
						<FontAwesomeIcon :icon="faSdCard" />
					</BaseButton>
					<input type="text" id="input-name" v-model="selected_element.name" placeholder="Anonym" @keydown.enter="submit" />
					<BaseButton
						v-if="reserved_elements[selected_element.mid] === undefined"
						@click="selected_element.name = undefined"
					>
						<FontAwesomeIcon :icon="faXmark" />
					</BaseButton>
					<BaseButton
						v-else
						@click="delete_reservation"
					>
						<FontAwesomeIcon :icon="faTrashCan" />
					</BaseButton>
				</template>
			</div>
		</BasePV>
		<AdminAccount v-else-if="window_state === WindowState.Account" />
		<AdminUsers v-else-if="window_state === WindowState.Users" />
	</AppLayout>
</template>

<style scoped>
	#main-view {
		width: 100%;
		height: 100%;
	}

	.navbar-item.active {
		text-decoration: underline;

		font-weight: bold;
	}

	#tooltip_content {
		display: flex;
		
		align-items: center;

		gap: 0.25em;
	}

	#input-name {
		transition: opacity 0.2s;

		width: 100%;
	}

	#input-name:disabled {
		cursor: not-allowed;
		opacity: 50%;
	}
</style>
