// Country names for the ISO codes teams and users carry (the importer's
// mapping, internal/importer/countries.go). The UK home nations use the
// flag-icons sub-national codes.

const SUB: Record<string, string> = {
	'gb-eng': 'England',
	'gb-sct': 'Scotland',
	'gb-wls': 'Wales',
	'gb-nir': 'Northern Ireland'
};

let names: Intl.DisplayNames | null = null;
try {
	names = new Intl.DisplayNames(undefined, { type: 'region' });
} catch {
	names = null;
}

/** "at" → "Austria" (in the device language). */
export function countryName(code: string): string {
	const c = code.toLowerCase();
	if (SUB[c]) return SUB[c];
	if (c === 'xk') return 'Kosovo';
	try {
		return names?.of(c.toUpperCase()) ?? c.toUpperCase();
	} catch {
		return c.toUpperCase();
	}
}

/** Football countries a user can pick in Settings, by name. */
const CODES = [
	// UEFA
	'al', 'ad', 'am', 'at', 'az', 'by', 'be', 'ba', 'bg', 'hr', 'cy', 'cz', 'dk', 'gb-eng', 'ee', 'fo', 'fi',
	'fr', 'ge', 'de', 'gi', 'gr', 'hu', 'is', 'il', 'it', 'kz', 'xk', 'lv', 'li', 'lt', 'lu', 'mt', 'md',
	'mc', 'me', 'nl', 'mk', 'gb-nir', 'no', 'pl', 'pt', 'ie', 'ro', 'ru', 'sm', 'gb-sct', 'rs', 'sk', 'si',
	'es', 'se', 'ch', 'tr', 'ua', 'gb-wls',
	// elsewhere
	'ar', 'bo', 'br', 'cl', 'co', 'ec', 'py', 'pe', 'uy', 've', 'ca', 'cr', 'mx', 'pa', 'us', 'jm',
	'au', 'cn', 'in', 'ir', 'iq', 'jp', 'kr', 'qa', 'sa', 'ae', 'uz', 'dz', 'cm', 'cd', 'eg', 'gh',
	'ci', 'ma', 'ng', 'sn', 'za', 'tn', 'nz'
];

export function countryOptions(): { code: string; name: string }[] {
	return CODES.map((code) => ({ code, name: countryName(code) })).sort((a, b) => a.name.localeCompare(b.name));
}
