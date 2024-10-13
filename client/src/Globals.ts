import { ref } from "vue";
import { api_call } from "./lib";

export type ReservedModules = Record<string, string>;

export const reserved_modules = ref<ReservedModules>({});

void (async () => {
	const reserved_modules_request = api_call<{ reserved_modules: ReservedModules}>("GET", "modules");

	if ((await reserved_modules_request).ok) {
		reserved_modules.value = (await (await reserved_modules_request).json()).reserved_modules
	}
})();

export interface User {
	uid: number;
	name: string;
}

export interface UserLogin extends User {
	logged_in: boolean;
}

export const user = ref<UserLogin>();

void (async () => {
	const response = await api_call<UserLogin>("GET", "welcome");

	if (response.ok) {
		user.value = await response.json();
	}
})()