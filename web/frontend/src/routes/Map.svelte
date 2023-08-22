<script lang="ts">
	import { onMount, onDestroy, afterUpdate } from 'svelte';
	import { Map, NavigationControl, Marker, Popup } from 'maplibre-gl';
	import distinctColors from 'distinct-colors';
	import 'maplibre-gl/dist/maplibre-gl.css';
	import Menu from './Menu.svelte';
	import Legend from './Legend.svelte';
	import { pilotsStore } from './stores';

	const apiKey: string = '2eGD0NL9iHVOoN7ymayK';

	if (!apiKey) {
		throw new Error('You need to configure env API_KEY first, see README');
	}

	let dates: string[] = []; // Possible dates to select, will be retrieved from the DB.
	let selectedDate: string = new Date().toISOString().slice(0, 10); // Default tracking date is today.

	async function getTracks() {
		try {
			const response = await fetch('http://localhost:8080/tracks/' + selectedDate, {
				method: 'GET',
				headers: {
					Accept: 'application/json'
				}
			});

			if (!response.ok) {
				throw new Error(`Error! status: ${response.status}`);
			}

			const result = await response.json();
			return result;
		} catch (err) {
			console.error(err);
		}
	}

	async function getFetch(url: string) {
		return await fetch(url).then((res) => {
			return res.json();
		});
	}

	const getDatesWithCount = () => {
		getFetch('http://localhost:8080/dates').then((res) => {
			dates = res['dates']
				.filter((v) => v.slice(0, 10) != new Date().toISOString().slice(0, 10))
				.map((v) => {
					return v.slice(0, 10);
				});
		});
	};

	let map: Map;

	// Keep track of the layers to clear and update them.
	let addedLayers: { layerId: string; marker?: Marker }[] = [];
	let center: [number, number] = [7.2869, 46.4718];

	const refreshMap = () => {
		addedLayers.forEach((layerInfo) => {
			const { layerId, marker } = layerInfo;

			if (map.getLayer(layerId)) {
				map.removeLayer(layerId);
				map.removeSource(layerId);
			}

			if (marker) {
				marker.remove();
			}
		});
		addedLayers = [];

		if (map) {
			updateMap();
		}
	};

	const updateMap = () => {
		getDatesWithCount();

		getTracks().then((data) => {
			pilotsStore.set([]);
			let tracks = Object.fromEntries(Object.entries(data).filter(([k, v]) => v.length > 0));
			console.log(tracks);
			const colors = distinctColors({ count: Object.keys(tracks).length });
			let i: number = 0;
			Object.entries(tracks).forEach(([pilot, track]) => {
				const coordinates: number[][] = [];
				const color = colors[i].hex();

				track.forEach((point) => {
					coordinates.push([point.Longitude, point.Latitude]);
				});

				const point = track[track.length - 1];
				// TODO: improve centering
				center = [point.Longitude, point.Latitude];
				// Add only the last point.
				const marker = new Marker({
					color: color,
					scale: 0.75
				})
					.setLngLat([point.Longitude, point.Latitude])
					.setPopup(
						new Popup({ offset: 25 }).setHTML(
							`<h3>${pilot}</h3>
              <p>Alt: ${point.Altitude} m<br\>
              Track: ${point.CumDist} km<br\>
              T.Off: ${point.TakeOffDist} km<br\>
              Msg: ${point.MsgContent}<br\>
              Time: ${point.DateTime}<br\>
              Lat: ${point.Latitude.toFixed(5)} Lng: ${point.Longitude.toFixed(5)}</p>`
						)
					)
					.addTo(map);
				//const markerElement = marker.getElement();
				//markerElement.textContent = (i + 1).toString();
				const layerId: string = `route${pilot}`;
				map.addLayer({
					id: layerId,
					type: 'line',
					source: {
						type: 'geojson',
						data: {
							type: 'Feature',
							geometry: {
								type: 'LineString',
								coordinates
							}
						}
					},
					layout: {
						'line-join': 'round',
						'line-cap': 'round'
					},
					paint: {
						'line-color': color,
						'line-width': 5
					}
				});
				addedLayers.push({ layerId, marker });
				i++;

				console.log(color);
				// Add the pilot in the list
				pilotsStore.update((existingPilots) => [
					...existingPilots,
					{
						color: color,
						name: pilot,
						altitude: point.Altitude,
						track: '',
						last: point.DateTime.slice(11, 16)
					}
				]);
			});
			console.log(pilotsStore);
		});
	};

	onMount(() => {
		map = new Map({
			container: 'map',
			//style: `https://api.maptiler.com/maps/streets/style.json?key=${apiKey}`,
			//style: `https://api.maptiler.com/maps/outdoor-v2/style.json?key=${apiKey}`,
			style: `https://api.maptiler.com/maps/topo-v2/style.json?key=${apiKey}`,
			//style: `https://api.maptiler.com/maps/backdrop/style.json?key=${apiKey}`,
			center: center,
			zoom: 10
		});
		map.addControl(new NavigationControl(), 'top-right');

		refreshMap();
		map.setCenter(center);
	});

	afterUpdate(() => {
		if (map) {
			map.setCenter(center);
		}
	});

	onDestroy(() => {
		if (map) {
			map.remove();
		}
	});

	$: if (selectedDate) refreshMap();
</script>

<section id="query-section">
	<Menu {dates} bind:selectedDate />
</section>

<section id="query-section">
	<Legend />
</section>

<div class="map-wrap">
	<a href="https://www.maptiler.com" class="watermark"
		><img src="https://api.maptiler.com/resources/logo.svg" alt="MapTiler logo" /></a
	>
	<div class="map" id="map" />
</div>

<style>
	.map-wrap {
		position: relative;
		width: 100%;
		height: 100vh;
	}

	.map {
		position: absolute;
		width: 100%;
		height: 100%;
	}

	.watermark {
		position: absolute;
		left: 10px;
		bottom: 10px;
		z-index: 999;
	}
</style>
