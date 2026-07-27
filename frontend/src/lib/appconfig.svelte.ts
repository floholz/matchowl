import { api } from './api';

// Deploy-time settings (Ko-Fi / contact links) fetched once per session from
// /api/appconfig. Consumers call load() on mount; repeat calls are no-ops.
class AppConfigStore {
	kofiUrl = $state('');
	contactEmail = $state('contact@floholz.dev'); // fallback until loaded
	loaded = $state(false);
	private loading = false;

	async load() {
		if (this.loaded || this.loading) return;
		this.loading = true;
		try {
			const r = await api.appConfig();
			this.kofiUrl = r.kofiUrl ?? '';
			if (r.contactEmail) this.contactEmail = r.contactEmail;
			this.loaded = true;
		} catch {
			/* keep fallbacks; retry on next load() call */
		} finally {
			this.loading = false;
		}
	}
}

export const appConfig = new AppConfigStore();
