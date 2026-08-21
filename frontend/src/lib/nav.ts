import { Newspaper, Trophy, Users } from '@lucide/svelte';
import type { Component } from 'svelte';

export interface NavItem {
	href: string;
	label: string;
	icon: Component;
}

export const navItems: NavItem[] = [
	{ href: '/', label: 'Feed', icon: Newspaper },
	{ href: '/competitions', label: 'Competitions', icon: Trophy },
	{ href: '/friends', label: 'Friends', icon: Users }
];

export function isActive(href: string, path: string): boolean {
	if (href === '/') return path === '/';
	// The forecast page is a per-season view; it belongs to Competitions.
	if (href === '/competitions')
		return path.startsWith('/competitions') || path.startsWith('/forecast');
	return path.startsWith(href);
}
