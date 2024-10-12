<script setup lang="ts">
	import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
	import { faPowerOff } from '@fortawesome/free-solid-svg-icons';

	import BaseButton from './BaseButton.vue';
	import { type UserLogin } from '@/AppAdmin.vue';
	import { api_call } from '@/lib';
	import type { User } from './AdminUsers.vue';

	const user = defineModel<UserLogin>();

	async function logout() {
		const response = await api_call<User>("GET", "logout");

		if (response.ok) {
			user.value = await response.json();
		}
	}
</script>

<template>
	<div id="navbar">
		<div id="center">
			<slot></slot>
		</div>
		<BaseButton @click="logout">
			<FontAwesomeIcon :icon="faPowerOff" />
		</BaseButton>
	</div>
</template>

<style scoped>
	#navbar {
		margin-bottom: 1em;
		
		display: flex;
	}
	
	#center {
		display: flex;
	
		align-items: baseline;
		justify-content: center;

		flex: 1;

		gap: 1em;
	}
</style>
