<!-- Match page /m/{id}: the MatchDetail component as a page (on desktop
     the Matches list shows the same component in its detail panel). -->
<script lang="ts">
	import { page } from '$app/stores';
	import { tournamentStore } from '$lib/tournament.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import MatchDetail from '$lib/components/MatchDetail.svelte';

	let id = $derived($page.params.id ?? '');
	let match = $derived(tipsStore.matches.find((m) => m.id === id));
	let season = $derived(tournamentStore.current);
	pageChrome(() => ({
		back: '/matches',
		title: (match && season?.competition?.name) || 'Match',
		context: match
			? [tournamentStore.stageName(match.stage), match.roundLabel !== tournamentStore.stageName(match.stage) ? match.roundLabel : '']
					.filter(Boolean)
					.join(' · ')
			: ''
	}));
</script>

<MatchDetail {id} />
