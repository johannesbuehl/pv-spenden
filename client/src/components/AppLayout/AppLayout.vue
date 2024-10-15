<script setup lang="ts">
import { user, type User } from '@/Globals';
import LayoutHeaderFooter from './LayoutHeaderFooter.vue';
import { api_call } from '@/lib';
import { faPowerOff } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';


	const footer_sites = {
		/* eslint-disable @typescript-eslint/naming-convention */
		About: "/about",
		Datenschutz: "/legal/datenschutz",
		Impressum: "/legal/impressum"
		/* eslint-enable @typescript-eslint/naming-convention */
	};

	function is_home(pathname: string): boolean {
		return window.location.pathname === pathname;
	}

	async function logout() {
		const response = await api_call<User>("GET", "logout");

		if (response.ok) {
			user.value = await response.json();
		}
	}
</script>

<template>
	<LayoutHeaderFooter v-if="user?.logged_in || !is_home('/')">
		<a v-if="!is_home('/')" href="/">Home</a>

		<template
			v-if="user?.logged_in"
		>
			<a v-if="!is_home('/admin.html')" href="/admin.html">Admin</a>
			
			<slot name="header"></slot>
		</template>
			<template #right
				v-if="user?.logged_in"
			>
				<a @click="logout"><FontAwesomeIcon :icon="faPowerOff" /></a>
			</template>
	</LayoutHeaderFooter>
	<div id="scroll">
		<div id="app_content">
			<slot></slot>
		</div>
	</div>
	<LayoutHeaderFooter id="footer">
		<a
			v-for="[name, url] in Object.entries(footer_sites)"
			:key="name"
			:href="url"
			:class="{ active: is_home(url) }"
		>
			{{ name }}
		</a>
	</LayoutHeaderFooter>
</template>

<style scoped>
	#scroll {
		width: 100%;
		height: 100%;

		overflow: auto;

		display: flex;
		justify-content: center;
	}

	#footer {
		margin-top: auto;

		font-size: 0.75em;
	}

	.active {
		font-weight: bold;

		text-decoration: underline;
	}
</style>

<style>
	#app {
		margin: 0 auto;
		padding: 0.25em;
		height: 100vh;
		width: 100vw;

		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5em;

		overflow: clip;
	}

	body {
		margin: 0;
	}
</style>
