<script setup lang="ts">
	import { ref } from "vue";
	import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
	import { faRightToBracket } from "@fortawesome/free-solid-svg-icons";

	import { api_call, HTTPStatus } from "@/lib";
	import type { UserLogin } from "@/Globals";
	
	import BaseButton from "./BaseButton.vue";

	const user_input = ref<string>("");
	const password_input = ref<string>("");
	const wrong_password = ref<boolean>(false);

	const user = defineModel<UserLogin>()

	async function login() {
		const response = await api_call<UserLogin>(
			"POST",
			"login",
			undefined,
			{ user: user_input.value, password: password_input.value }
		);

		if (response.ok) {
			wrong_password.value = false;

			const response_data = await response.json();

			user.value = response_data;
		} else {
			if (response.status === HTTPStatus.Unauthorized) {
				wrong_password.value = true;
			}
		}
	}
</script>

<template>
	<div id="content">
		<div v-if="wrong_password" id="wrong-password">
			<h2>Login fehlgeschlagen</h2>
			unbekannter Benutzer oder fasches Passwort
		</div>
		<form id="login">
			<input
				id="username"
				type="text"
				name="name"
				autocomplete="username"
				:required="true"
				v-model="user_input"
				placeholder="Name"
				@keydown.enter="login"
			/>
			<input
				id="password"
				type="password"
				name="password"
				autocomplete="current-password"
				:required="true"
				v-model="password_input"
				placeholder="Passwort"
				@keydown.enter="login"
			/>
			<BaseButton @click="login"
				><FontAwesomeIcon :icon="faRightToBracket" /> Login</BaseButton
			>
		</form>
	</div>
</template>

<style scoped>
	#content {
		margin-inline: auto;

		display: flex;
		flex-direction: column;
		gap: 0.25em;

		max-width: 15em;
		height: 100%;

		justify-content: center;
	}

	#wrong-password {
		color: red;
	}

	#login {
		width: 100%;

		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25em;
	}

	#login input {
		width: 100%;
	}
</style>