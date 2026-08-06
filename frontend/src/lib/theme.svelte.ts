import { browser } from '$app/environment';

export type ThemeMode = 'system' | 'light' | 'dark' | 'amoled';

const STORAGE_KEY = 'matchowl-theme';

/** The three Matchowl themes plus "follow the device". Applies data-theme on
 *  <html> (theme.css keys off it) and keeps the PWA theme-color meta in sync
 *  so the browser chrome matches. */
class ThemeStore {
	mode = $state<ThemeMode>('system');

	readonly modes: { value: ThemeMode; label: string; hint: string }[] = [
		{ value: 'system', label: 'Auto', hint: 'Follow this device' },
		{ value: 'light', label: 'Light', hint: 'Peach paper' },
		{ value: 'dark', label: 'Dark', hint: 'Warm night' },
		{ value: 'amoled', label: 'Black', hint: 'True black for OLED screens' }
	];

	init() {
		if (!browser) return;
		const saved = localStorage.getItem(STORAGE_KEY) as ThemeMode | null;
		if (saved && this.modes.some((m) => m.value === saved)) this.mode = saved;
		this.apply();
		// Follow device switches while in system mode.
		window
			.matchMedia('(prefers-color-scheme: light)')
			.addEventListener('change', () => {
				if (this.mode === 'system') this.apply();
			});
	}

	set(mode: ThemeMode) {
		this.mode = mode;
		if (browser) localStorage.setItem(STORAGE_KEY, mode);
		this.apply();
	}

	/** The theme actually rendered (resolves 'system'). */
	get resolved(): Exclude<ThemeMode, 'system'> {
		if (this.mode !== 'system') return this.mode;
		if (browser && window.matchMedia('(prefers-color-scheme: light)').matches)
			return 'light';
		return 'dark';
	}

	private apply() {
		if (!browser) return;
		document.documentElement.dataset.theme = this.resolved;
		// Keep the browser/PWA chrome color in step with the theme.
		requestAnimationFrame(() => {
			const c = getComputedStyle(document.documentElement)
				.getPropertyValue('--theme-color')
				.trim();
			if (!c) return;
			document
				.querySelector('meta[name="theme-color"]')
				?.setAttribute('content', c);
		});
	}
}

export const theme = new ThemeStore();
