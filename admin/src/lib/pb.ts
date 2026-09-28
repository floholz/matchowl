import PocketBase from 'pocketbase';
import { browser } from '$app/environment';

// Same origin: the Go binary serves the API on the admin host too, and in
// dev Vite proxies /api to the local backend. The admin origin has its own
// localStorage, so an admin signs in here separately from the player app.
export const pb = new PocketBase(browser ? window.location.origin : '/');
pb.autoCancellation(false);
