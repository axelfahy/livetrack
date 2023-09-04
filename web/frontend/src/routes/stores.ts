import { writable } from 'svelte/store';

interface Pilot {
	color: string;
	name: string;
	altitude: number;
	track: string;
	last: string;
}

export const pilotsStore = writable<Pilot[]>([]);
