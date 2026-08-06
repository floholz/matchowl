import { Newspaper, Volleyball, Trophy } from '@lucide/svelte';
import type { Component } from 'svelte';

export interface NavItem {
	href: string;
	label: string;
	icon: Component;
}

export const navItems: NavItem[] = [
	{ href: '/', label: 'Feed', icon: Newspaper },
	{ href: '/tournaments', label: 'Tournaments', icon: Volleyball },
	{ href: '/leagues', label: 'Leagues', icon: Trophy }
];

export function isActive(href: string, path: string): boolean {
	if (href === '/') return path === '/';
	// The per-tournament detail pages (/tips, /forecast, /tournament) belong
	// to the Tournaments destination.
	if (href === '/tournaments')
		return (
			path.startsWith('/tournaments') ||
			path.startsWith('/tips') ||
			path.startsWith('/forecast') ||
			path.startsWith('/tournament')
		);
	return path.startsWith(href);
}
