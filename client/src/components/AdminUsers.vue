<script lang="ts">
	export function validate_password(password: string): boolean {
		const password_length = password?.length;

		return password_length >= 12 && password_length <= 64;
	}
</script>

<script setup lang="ts">
	import { faPlus, faSdCard, faTrash } from '@fortawesome/free-solid-svg-icons';
	import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
	import { onMounted, ref } from 'vue';

	import BaseButton from './BaseButton.vue';
	import { api_call } from '@/lib';
	import type { User } from '@/Globals';

	interface PasswordUser extends User {
		password: string
	}

	const add_user_name_input = ref<string>("");
	const add_user_password_input = ref<string>("");
	const users = ref<PasswordUser[]>([]);

	onMounted(async () => {
		const response = await api_call<User[]>("GET", "users");

		if (response.ok) {
			store_users(await response.json());
		}
	});

	function store_users(new_user: User[]) {
		users.value = new_user.map((user) => {
			return { ...user, password: "" };
		});
	}

	function validate_new_user(): boolean {
		if (add_user_name_input.value.length === 0) {
			return false;
		}

		if (!validate_password(add_user_password_input.value)) {
			return false;
		}

		return true;
	}

	async function add_user() {
		if (validate_new_user()) {
			const response = await api_call<User[]>("POST", "users", undefined, {
				name: add_user_name_input.value,
				password: add_user_password_input.value
			});

			if (response.ok) {
				store_users(await response.json())
			}
		}
	}

	async function delete_user(user: PasswordUser) {
		if (user.name !== "admin") {
			if (window.confirm(`Delete user '${user.name}'?`)) {
				const response = await api_call<User[]>("DELETE", "users", { uid: user.uid });

				if (response.ok) {
					store_users(await response.json());
				}
			}
		}
	}

	async function modify_user(user: PasswordUser) {
		if (validate_password(user.password)) {
			const response = await api_call<User[]>("PATCH", "users", { uid: user.uid }, { password: user.password });

			if (response.ok) {
				users.value = await response.json();
			}
		}
	}
</script>

<template>
	<div id="admin-wrapper">
		<h1>Benutzer</h1>
		<div id="add-user-wrapper">
			<div id="add-user-wrapper-inputs">
				Benutzername: <input type="text" v-model="add_user_name_input" placeholder="username" />
				Passwort: <input type="text" v-model="add_user_password_input" placeholder="password" />
			</div>
			<BaseButton :disabled="!validate_new_user()" @click="add_user">
				<FontAwesomeIcon :icon="faPlus" />
			</BaseButton>
		</div>
		<div id="modify-user-wrapper">
			<table id="users">
				<thead>
					<tr class="header">
						<th>UID</th>
						<th>Name</th>
						<th>Passwort</th>
						<th>Bestätigen</th>
						<th>Löschen</th>
					</tr>
				</thead>
				<tbody>
					<tr class="content" v-for="user of users" :key="user.uid">
						<th>{{ user.uid }}</th>
						<th>{{ user.name }}</th>
						<th>
							<div class="cell">
								<input type="text" v-model="user.password" placeholder="Neues Passwort" />
							</div>
						</th>
						<th>
							<div class="cell">
								<BaseButton class="button" :disabled="!validate_password(user.password)" @click="modify_user(user)"
									><FontAwesomeIcon :icon="faSdCard"
								/></BaseButton>
							</div>
						</th>
						<th>
							<div class="cell">
								<BaseButton
									class="button"
									:disabled="user.name === 'admin'"
									@click="delete_user(user)"
									><FontAwesomeIcon :icon="faTrash"
								/></BaseButton>
							</div>
						</th>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
</template>

<style scoped>
	#admin-wrapper {
		display: flex;
		flex-direction: column;

		align-items: center;
		gap: 0.5em;
	}

	#add-user-wrapper {
		display: flex;
		align-items: center;

		gap: 1em;
	}

	#add-user-wrapper-inputs {
		display: grid;

		grid-template-columns: auto auto;

		column-gap: 0.5em;
	}
	
	#users {
		max-width: 40em;
	}

	tr.header * {
		font-weight: bolder;

		background-color: black;
		color: white;
	}

	tr.content:nth-of-type(2n) {
		background-color: hsl(0, 0%, 90%);
	}

	tr.content:nth-of-type(2n + 1) {
		background-color: hsl(0, 0%, 80%);
	}

	th {
		padding: 0.25em;
	}

	th > div.cell {
		width: 100%;

		display: flex;
		align-items: center;
		justify-content: center;
	}

	th input[type="text"] {
		flex: 1;
	}

	tr.content input[type="text"] {
		font-size: 0.67em;
	}
</style>
