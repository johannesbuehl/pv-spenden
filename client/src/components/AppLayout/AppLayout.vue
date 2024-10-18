<script setup lang="ts">
	import { faBars, faPowerOff } from '@fortawesome/free-solid-svg-icons';
	import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';

	import { user, type User } from '@/Globals';
	import { api_call } from '@/lib';

	import LayoutHeaderFooter from './LayoutHeaderFooter.vue';
	import BaseButton from '../BaseButton.vue';
	import { ref } from 'vue';

	const hamburger_menu = ref<boolean>(false);

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
	<LayoutHeaderFooter v-if="user?.logged_in || !is_home('/')" id="header">
		<template #left>
			<BaseButton id="hamburger-toggle" :class="{ active: hamburger_menu }" @click="hamburger_menu = !hamburger_menu"><FontAwesomeIcon :icon="faBars" /></BaseButton>
		</template>

		<div
			id="header-content"
			:class="{ visible: hamburger_menu }"
		>
			<a v-if="!is_home('/')" href="/">Home</a>
			
			<template
				v-if="user?.logged_in"
			>
				<a v-if="!is_home('/admin.html')" href="/admin.html">Admin</a>
			
				<slot name="header"></slot>
			</template>
		</div>

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
		flex: 1;

		height: 100%;

		display: flex;
		justify-content: center;
	}

	#hamburger-toggle {
		display: none;

		transition: transform 0.5s ease;
	}

	#hamburger-toggle.active {
		transform: rotate(90deg);
	}

	#header-content {
		display: flex;

		align-items: baseline;

		column-gap: 2em;
	}

	@media screen and (max-width: 600px) {
		#header-content {
			flex-direction: column;
			align-items: center;
		}

		#header-content:not(.visible) {
			display: none;
		}

		#hamburger-toggle {
			display: unset;
		}
	}

	#footer {
		margin-top: auto;

		font-size: 0.75em;
	}

	@media screen and (max-width: 400px) {
		#footer:deep(div) {
			flex-direction: column;

			align-items: center;
		}
	}

	.active {
		font-weight: bold;

		text-decoration: underline;
	}
</style>

<style>
	#app {
		padding: 0.25em;

		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5em;
		
		min-height: 100cqh;
	}
</style>
