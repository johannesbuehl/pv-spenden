<script lang="ts">
	export function validate_password(password_new: string, password_repeat?: string, password_old?: string): string[]{
		const result: string[] = [];

		if (password_old !== undefined && password_old.length === 0) {
			result.push("Bisheriges Passwort fehlt");
		}

		if (password_new.length < 12) {
			result.push("Passwort muss mindestens 12 Zeichen lang sein");
		}

		if (password_new.length > 64) {
			result.push("Passwort darf höchstens 64 Zeichen lang sein");
		}

		if (password_repeat !== undefined && password_new !== password_repeat) {
			result.push("Passwörter stimmen nicht überein");
		}

		return result;
	}
</script>

<script setup lang="ts">
	import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
	import BaseButton from './BaseButton.vue';
	import { faSdCard } from '@fortawesome/free-solid-svg-icons';
	import { ref } from 'vue';
	
	import { api_call } from '@/lib';

	const password_current = ref<string>("");
	const password_new = ref<string>("");
	const password_repeat = ref<string>("");

	async function change_password() {
		if (validate_password(password_new.value, password_repeat.value, password_current.value).length === 0) {
			const response = await api_call<{}>("PATCH", "user/password", undefined, {
				password: password_new.value
			});

			if (response.ok) {
				alert("Passwort erfolgreich geändert");
				password_current.value = "";
				password_new.value = "";
				password_repeat.value = "";
			}
		}
	}
</script>

<template>
	<div id="account-wrapper">
		<h1>Account</h1>
		<div id=change-password>
			<form id="change-password-inputs">
				<input style="display: none;" type="text" autocomplete="username">
				Bisheriges Passwort
				<input type="password" autocomplete="current-password" v-model="password_current">
				Neues Passwort
				<input type="password" autocomplete="new-password" v-model="password_new">
				Neues Passwort wiederholen
				<input type="password" autocomplete="new-password" v-model="password_repeat">
			</form>
			<div
				v-if="validate_password(password_new, password_repeat, password_current).length > 0"
				id="password-error-text"
			>
				<div
					v-for="e in validate_password(password_new, password_repeat, password_current)"
					:key="e"
				>
					{{ e }}
				</div>
			</div>
			<BaseButton id="btn-change-password" :disabled="validate_password(password_new, password_repeat, password_current).length > 0"  @click="change_password"><FontAwesomeIcon :icon="faSdCard" /> Passwort ändern</BaseButton>
		</div>
	</div>
</template>

<style scoped>
	#account-wrapper {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	#change-password {
		display: flex;
		flex-direction: column;

		align-items: center;

		gap: 1em;
	}

	#change-password-inputs {
		display: grid;

		grid-template-columns: auto auto;

		column-gap: 0.5em;
	}

	#password-error-text {
		color: darkorange;
	}

	#btn-change-password {
		padding: 0.25em;
		
		border: 0.0625em solid black;
		border-radius: 0.125em;
	}
</style>
