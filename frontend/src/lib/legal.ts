// Legal acceptance. Bump TERMS_VERSION when the terms or the privacy notice
// change in a way people must accept again: every signed-in account whose
// stored termsVersion differs is sent to /accept-terms by the layout guard.
export const TERMS_VERSION = '2026-09-13';

export function termsCurrent(v: string | undefined | null): boolean {
	return v === TERMS_VERSION;
}
