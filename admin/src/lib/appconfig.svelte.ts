import { api } from './api';

// Deploy-time settings from /api/appconfig, fetched once per session.
class AppConfigStore {
	version = $state('');
	registrationOpen = $state(true);
	/** The player app's public origin (mail links, invite links). */
	appUrl = $state('');
	loaded = $state(false);
	private loading = false;

	async load() {
		if (this.loaded || this.loading) return;
		this.loading = true;
		try {
			const r = await api.appConfig();
			this.version = r.version ?? '';
			this.registrationOpen = r.registrationOpen !== false;
			this.appUrl = r.appUrl ?? '';
			this.loaded = true;
		} catch {
			/* keep fallbacks; retry on next load() call */
		} finally {
			this.loading = false;
		}
	}
}

export const appConfig = new AppConfigStore();
