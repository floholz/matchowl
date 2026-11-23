import { House, List, Trophy, Users } from '@lucide/svelte';
import type { Component } from 'svelte';

export interface NavItem {
	href: string;
	label: string;
	icon: Component;
}

/** Pools are the center of the app (PLAN.md, "pools at the center"):
 *  Competitions is no tab; its pages hang off Matches. */
export const navItems: NavItem[] = [
	{ href: '/', label: 'Home', icon: House },
	{ href: '/matches', label: 'Matches', icon: List },
	{ href: '/pools', label: 'Pools', icon: Trophy },
	{ href: '/friends', label: 'Friends', icon: Users }
];

export function isActive(href: string, path: string): boolean {
	if (href === '/') return path === '/';
	// Match and competition pages belong to Matches.
	if (href === '/matches')
		return path.startsWith('/matches') || path.startsWith('/m/') || path.startsWith('/competitions');
	// Forecasts belong to classic pools.
	if (href === '/pools') return path.startsWith('/pools') || path.startsWith('/forecast');
	return path.startsWith(href);
}
