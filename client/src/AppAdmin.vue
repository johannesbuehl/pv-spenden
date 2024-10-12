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
	import { faPlus, faSdCard, faTrashCan, faXmark } from '@fortawesome/free-solid-svg-icons';
	import { onMounted, ref, watch } from 'vue';
	
	import BasePV, { type Module } from './components/BasePV.vue';
	import { api_call, type APICallResult } from './lib';
	import { reserved_modules, type ReservedModules } from './Globals';
	import AdminUsers, { type User } from './components/AdminUsers.vue';
	import BaseButton from './components/BaseButton.vue';
	import AdminLogin from './components/AdminLogin.vue';
	import AdminNavbar from './components/AdminNavbar.vue';
	import AdminAccount from './components/AdminAccount.vue';

	const window_state = ref<WindowState>(WindowState.Login);
	const user = ref<UserLogin>();
	const selected_module = ref<Module>();

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
		// check wether a module is selected
		if (selected_module.value !== undefined) {
			let response: APICallResult<ReservedModules>;
			
			const name = selected_module.value.name !== "" ? selected_module.value.name : "Anonym";
				
			// if the module is already reserved, patch it instead
			const method = reserved_modules.value[selected_module.value.mid] === undefined ? "POST" : "PATCH";
				
			response = await api_call<{ reserved_modules: ReservedModules }>(method, "modules", { mid: selected_module.value }, {
				name
			});
			
			if (response.ok) {
				reserved_modules.value = (await response.json()).reserved_modules;
				
				selected_module.value = undefined;
			} else {
				alert(`Error during database write: ${await response.text()}`);
			}
		}
	}

	async function delete_reservation() {
		// only proceed if there is a valid module-selection
		if (selected_module.value !== undefined) {
			const module_name = selected_module.value?.mid.match(/\w\d+/)?.["0"].toUpperCase()
			
			if (confirm(`Delete reservation for module "${module_name}" with the name "${reserved_modules.value[selected_module.value.mid]}"?`)) {
				const response = await api_call<{ reserved_modules: ReservedModules }>("DELETE", "modules", { mid: selected_module.value.mid });
				
				if (response.ok) {
					reserved_modules.value = (await response.json()).reserved_modules;
				}
			}
		}
	}
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
			<div
				v-if="selected_module"
				id="tooltip_content"
			>
				<BaseButton
					v-if=" selected_module.name === undefined"
					@click="selected_module.name = ''"
				>
					<FontAwesomeIcon :icon="faPlus"></FontAwesomeIcon> Reserve Module
				</BaseButton>
				<template v-else>
					<BaseButton @click="submit">
						<FontAwesomeIcon :icon="faSdCard" />
					</BaseButton>
					<input type="text" id="input-name" v-model="selected_module.name" placeholder="Anonym" @keydown.enter="submit" />
					<BaseButton
						v-if="reserved_modules[selected_module.mid] === undefined"
						@click="selected_module.name = undefined"
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
	</div>
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
	}

	#input-name:disabled {
		cursor: not-allowed;
		opacity: 50%;
	}
</style>
