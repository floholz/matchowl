<!-- Delivery policy: the per-channel kill switches and per-event overrides
     for every automated mail and push. Its own page — too important to
     sit under a growing announcements list. -->
<script lang="ts">
	import NotifyPolicyCard from '$lib/components/NotifyPolicyCard.svelte';
</script>

<div class="head">
	<div>
		<p class="kicker">Admin</p>
		<h1>Notifications</h1>
	</div>
</div>

<NotifyPolicyCard />

<style>
	.head {
		margin-bottom: 1rem;
	}
</style>
