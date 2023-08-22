<script lang="ts">
	export let dates;
	export let selectedDate;
	$: console.log('Dates:', dates, 'Selected:', selectedDate);
</script>

<section class="menu-dates">
	<select class="menu" name="menu" id="menu" bind:value={selectedDate}>
		<option selected value={new Date().toISOString().slice(0, 10)}>Today</option>
		{#each dates as date}
			<option value={date}>{date}</option>
		{/each}
	</select>
</section>

<style>
	.menu-dates {
		position: absolute;
		display: inline-block;
		top: 20px;
		left: 20px;
		z-index: 1000;
	}

	select {
		margin: 0;
		font-size: 1rem;
		border: 1px solid gray;
		border-radius: 5px;
	}

	.menu {
		width: 100%;
	}
</style>
