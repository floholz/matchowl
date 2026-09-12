import { House, List, Trophy, Users } from '@lucide/svelte';
import type { Component } from 'svelte';

export interface NavItem {
	href: string;
	label: string;
	icon: Component;
}

export const navItems: NavItem[] = [
	{ href: '/', label: 'Home', icon: House },
	{ href: '/matches', label: 'Matches', icon: List },
	{ href: '/competitions', label: 'Competitions', icon: Trophy },
	{ href: '/friends', label: 'Friends', icon: Users }
];

export function isActive(href: string, path: string): boolean {
	if (href === '/') return path === '/';
	// Match pages belong to Matches.
	if (href === '/matches') return path.startsWith('/matches') || path.startsWith('/m/');
	// The forecast page is a per-season view; it belongs to Competitions.
	if (href === '/competitions')
		return path.startsWith('/competitions') || path.startsWith('/forecast');
	if (href === '/friends') return path.startsWith('/friends') || path.startsWith('/pools');
	return path.startsWith(href);
}
