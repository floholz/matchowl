<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	$effect(() => {
		goto(`/pools/${$page.params.id}/chat`, { replaceState: true });
	});
</script>
