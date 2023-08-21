<script>
  import Map from './Map.svelte';

  let mapComponent;

</script>

<div>
  <Map bind:this={mapComponent} />
</div>

<style>
  /* Page-specific styles... */
</style>