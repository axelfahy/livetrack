<script lang="ts">
	import { pilotsStore } from './stores';

	$: pilots = $pilotsStore;
	$: console.log('Pilots:', pilots);
</script>

<div class="legend">
	<table>
		<tr>
			<th />
			<th>Name</th>
			<th>Alt</th>
			<th>Trk/T.Off/Air Time</th>
			<th>Last</th>
		</tr>
		{#each pilots as pilot}
			<tr>
				<td>
					<div class="color-pilot" style="background-color: {pilot.color};" />
				</td>
				<td>{pilot.name}</td>
				<td>{pilot.altitude}</td>
				<td>{pilot.cumDist}/{pilot.takeOffDist}/{pilot.flightTime}</td>
				<td>{pilot.last}</td>
			</tr>
		{/each}
	</table>
</div>

<style>
	.legend {
		position: absolute;
		display: inline-block;
		z-index: 1000;
		top: 50px;
		left: 20px;
		background-color: rgba(255, 255, 255, 0.8);
		padding: 10px;
		border-radius: 5px;
	}

	.color-pilot {
		width: 20px;
		height: 20px;
		display: inline-block;
		margin-right: 10px;
		border: 1px solid #000;
	}

	table {
		width: 100%;
	}

	th,
	td {
		padding: 5px;
	}
</style>
