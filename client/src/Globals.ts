import { ref } from "vue";
import { api_call } from "./lib";

export type ReservedElements = Record<string, string>;

export const reserved_elements = ref<ReservedElements>({});

void (async () => {
	const reserved_elements_request = api_call<{ reserved_elements: ReservedElements}>("GET", "elements");

	if ((await reserved_elements_request).ok) {
		reserved_elements.value = (await (await reserved_elements_request).json()).reserved_elements
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