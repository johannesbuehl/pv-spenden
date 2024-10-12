<script lang="ts">
	export interface UserLogin extends User {
		logged_in: boolean;
	}

	enum WindowState {
		Login,
		Modules,
		Account,
		Users
	}
</script>

<script setup lang="ts">
	import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
	import { faSdCard, faTrashCan } from '@fortawesome/free-solid-svg-icons';
	import { onMounted, ref, watch } from 'vue';
	
	import BasePV from './components/BasePV.vue';
	import { api_call, type APICallResult } from './lib';
	import { reserved_modules, type ReservedModules } from './Globals';
	import AdminUsers, { type User } from './components/AdminUsers.vue';
	import BaseButton from './components/BaseButton.vue';
	import AdminLogin from './components/AdminLogin.vue';
	import AdminNavbar from './components/AdminNavbar.vue';
import AdminAccount from './components/AdminAccount.vue';

	const window_state = ref<WindowState>(WindowState.Login);
	const user = ref<UserLogin>();
	const input_name = ref<string>("");
	const selected_module = ref<string>("");

	onMounted(async () => {
		const response = await api_call<UserLogin>("GET", "welcome");

		if (response.ok) {
			user.value = await response.json();
		}
	});

	watch(user, user => {
			window_state.value = user?.logged_in ? WindowState.Modules : WindowState.Login
	}, { deep: true })

	async function submit() {
		// if the module is not reserved, send a delete-request
		let response: APICallResult<ReservedModules>;
		const name = input_name.value !== "" ? input_name.value : "Anonym";
		
		const method = reserved_modules.value[selected_module.value] === undefined ? "POST" : "PATCH";

		response = await api_call<{ reserved_modules: ReservedModules }>(method, "modules", { mid: selected_module.value }, {
			name
		});

		if (response.ok) {
			reserved_modules.value = (await response.json()).reserved_modules;

			selected_module.value = "";
		} else {
			alert(`Error during database write: ${await response.text()}`);
		}
	}

	async function delete_reservation() {
		const module_name = selected_module.value.match(/\w\d+/)?.["0"].toUpperCase()

		if (confirm(`Delete reservation for module "${module_name}" with the name "${reserved_modules.value[selected_module.value]}"?`)) {
			const response = await api_call<{ reserved_modules: ReservedModules }>("DELETE", "modules", { mid: selected_module.value });

			if (response.ok) {
				reserved_modules.value = (await response.json()).reserved_modules;
			}
		}
	}

	// watch the selected pv-module and handle the popup
	watch(selected_module, selected_module => {
		// clear the text
		input_name.value = "";

		// if a module is selected, load it
		if (selected_module) {
			const reserved_module_text = reserved_modules.value[selected_module];

			if (reserved_module_text !== undefined) {
				if (reserved_module_text === "") {
					input_name.value = "Anonym";
				} else {
					input_name.value = reserved_modules.value[selected_module];
				}
			}
		}
	});
</script>

<template>
	<AdminLogin v-if="window_state === WindowState.Login" v-model="user" />
	<div id="main-view" v-else>
		<AdminNavbar v-model="user">
			<BaseButton class="navbar-item" :class="{ active: window_state === WindowState.Modules }" @click="window_state = WindowState.Modules">Modules</BaseButton>
			<BaseButton class="navbar-item" :class="{ active: window_state === WindowState.Account }" @click="window_state = WindowState.Account">Account</BaseButton>
			<BaseButton v-if="user?.name === 'admin'" class="navbar-item" :class="{ active: window_state === WindowState.Users }" @click="window_state = WindowState.Users">Users</BaseButton>
		</AdminNavbar>
		<BasePV
			v-if="window_state === WindowState.Modules"
			v-model:selected_module="selected_module"
			>
			<div id="tooltip_content">	
				<input type="text" id="input-name" v-model="input_name" placeholder="Anonym" @keydown.enter="submit" />
				<BaseButton @click="submit">
					<FontAwesomeIcon :icon="faSdCard" />
				</BaseButton>
				<BaseButton
					v-if="reserved_modules[selected_module]"
					@click="delete_reservation"
				>
					<FontAwesomeIcon :icon="faTrashCan" />
				</BaseButton>
			</div>
		</BasePV>
		<AdminAccount v-else-if="window_state === WindowState.Account" />
		<AdminUsers v-else-if="window_state === WindowState.Users" />
	</div>
</template>

<style scoped>
	#main-view {
		width: 100%;
		height: 100%;

		padding: 1em;
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
	}

	#input-name:disabled {
		cursor: not-allowed;
		opacity: 50%;
	}
</style>
