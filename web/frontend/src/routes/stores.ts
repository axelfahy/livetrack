import { writable } from 'svelte/store';

interface Pilot {
	color: string;
	name: string;
	altitude: number;
	cumDist: number;
	takeOffDist: number;
	flightTime: string;
	last: string;
}

export const pilotsStore = writable<Pilot[]>([]);
