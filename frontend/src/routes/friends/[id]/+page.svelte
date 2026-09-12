<!-- Leagues became pools: keep old links (and the notifications that
     carry them) working. -->
<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	$effect(() => {
		goto(`/pools/${$page.params.id}`, { replaceState: true });
	});
</script>
