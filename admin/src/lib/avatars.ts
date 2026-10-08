// Drawn profile pictures: a template (the art) coloured by a palette. A
// user's pick is stored as "<template>:<palette>" in users.avatarPreset;
// nothing is rendered to an image, so retouching the art or a palette
// updates everyone. A picked preset is shown over an uploaded (or Google)
// photo, which stays stored so the user can switch back; an empty preset
// shows the photo, and without one the raven in a palette from the user
// id (stable across renames).
//
// Templates paint with three roles, as CSS variables on the <svg>:
//   --pp-h highlight (the background), --pp-a accent, --pp-d dark.
// A copy of frontend/src/lib/avatars.ts — keep the two in step.

export interface Palette {
	id: string;
	name: string;
	h: string;
	a: string;
	d: string;
}

export interface Template {
	id: string;
	name: string;
	/** Inner markup of a 256×256 viewBox. */
	svg: string;
}

const H = 'fill="var(--pp-h)"';
const A = 'fill="var(--pp-a)"';
const D = 'fill="var(--pp-d)"';

export const TEMPLATES: Template[] = [
	{
		// The raven from the naming days: a pitch on its chest.
		id: 'raven',
		name: 'Raven',
		svg: `<rect width="256" height="256" ${H}/>
<path ${D} d="M60.65 210.66C70.32 162.33 97.41 77.19 138.51 81.98C191.61 88.17 140.14 188.18 130.04 210.66H60.65Z"/>
<path ${A} fill-rule="evenodd" d="M166.85 72.53V210.66H60V72.53H166.85ZM158.91 202.71V145.57H130.69C128.88 153.44 121.84 159.32 113.43 159.32C105.01 159.32 97.97 153.44 96.17 145.57H67.94V202.71H88.09V180.71H138.76V202.71H158.91ZM158.91 80.48V137.62H130.69C128.88 129.75 121.84 123.87 113.43 123.87C105.01 123.87 97.97 129.75 96.17 137.62H67.94V80.48H88.09V102.48H138.76V80.48H158.91ZM96.02 94.53H130.83V80.48H96.02V94.53ZM130.83 188.65V202.71H96.02V188.65H130.83Z"/>
<path ${D} fill-rule="evenodd" d="M196.17 58.52C196.17 59.05 196.14 59.57 196.09 60.09C211.01 76.35 211.16 118.79 167.18 210.66H137.53C150.2 182.44 187.25 96.51 151.54 79.37C143.4 75.46 131.67 73.83 92.58 83.61C108.44 58.56 123.73 50.25 145.4 48.69C149.99 42.32 158.89 38 169.13 38C184.06 38 196.17 47.19 196.17 58.52Z"/>
<circle ${A} cx="161.64" cy="58.14" r="6.93"/>`
	}
];

export const PALETTES: Palette[] = [
	{ id: 'ember', name: 'Ember', h: '#FFD1A8', a: '#FF7700', d: '#000000' },
	{ id: 'mint', name: 'Mint', h: '#BDEFD8', a: '#0E9F6E', d: '#0B1F17' },
	{ id: 'sky', name: 'Sky', h: '#C7DDFF', a: '#2F6BFF', d: '#0B1430' },
	{ id: 'lilac', name: 'Lilac', h: '#E2D4FF', a: '#7B3FE4', d: '#1A1033' },
	{ id: 'rose', name: 'Rose', h: '#FFC9D6', a: '#E5336B', d: '#2A0B14' },
	{ id: 'lemon', name: 'Lemon', h: '#FFF0A6', a: '#E08A00', d: '#1F1A05' },
	{ id: 'clay', name: 'Clay', h: '#EADBC8', a: '#A0522D', d: '#1C140C' },
	// The classic colours inverted: a white raven at night.
	{ id: 'coal', name: 'Coal', h: '#000000', a: '#FF7700', d: '#FFFFFF' },
	// The WM-Tips colours, light and dark.
	{ id: 'lime', name: 'Lime', h: '#F3F5FB', a: '#C8FB50', d: '#10131F' },
	{ id: 'turf', name: 'Turf', h: '#10131F', a: '#91CF00', d: '#F3F5FB' }
];

/** Small stable hash of a string (FNV-1a). */
function hash(s: string): number {
	let x = 0x811c9dc5;
	for (let i = 0; i < s.length; i++) {
		x ^= s.charCodeAt(i);
		x = Math.imul(x, 0x01000193);
	}
	return x >>> 0;
}

/** Whether a stored preset names a known template and palette (one that
 *  was retired falls back to the photo or the default). */
export function knownPreset(preset: string | null | undefined): boolean {
	const [t, p] = (preset ?? '').split(':');
	return TEMPLATES.some((x) => x.id === t) && PALETTES.some((x) => x.id === p);
}

/** The template and palette for a user: their pick when it names a known
 *  one, else the raven in a palette chosen by the seed (the user id). */
export function resolvePreset(preset: string | null | undefined, seed: string): { template: Template; palette: Palette } {
	const [t, p] = (preset ?? '').split(':');
	const template = TEMPLATES.find((x) => x.id === t);
	const palette = PALETTES.find((x) => x.id === p);
	if (template && palette) return { template, palette };
	return { template: TEMPLATES[0], palette: PALETTES[hash(seed || '?') % PALETTES.length] };
}

/** The preset id stored for a template + palette. */
export const presetId = (template: string, palette: string) => `${template}:${palette}`;
