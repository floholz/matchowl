import { pb } from './pb';

// Reactive auth state backed by PocketBase's authStore. The admin app only
// cares about who is signed in and their role; the server enforces every
// admin endpoint on its own.
class Auth {
	user = $state<{
		id: string;
		name: string;
		email: string;
		avatarUrl: string | null;
		role: string; // "owner" | "admin" | "bot"; empty => normal member
	} | null>(null);

	constructor() {
		this.sync();
		pb.authStore.onChange(() => this.sync());
	}

	private sync() {
		const r = pb.authStore.record;
		if (!pb.authStore.isValid || !r) {
			this.user = null;
			return;
		}
		this.user = {
			id: r.id,
			name: (r.name as string) || r.email,
			email: r.email,
			avatarUrl: r.avatar ? pb.files.getURL(r, r.avatar as string) : null,
			role: (r.role as string) || 'member'
		};
	}

	get isAuthed() {
		return this.user !== null;
	}
	/** App-level admin (owner inherits admin). */
	get isAdmin() {
		return this.user?.role === 'admin' || this.user?.role === 'owner';
	}
	get isOwner() {
		return this.user?.role === 'owner';
	}

	async login(identity: string, password: string) {
		await pb.collection('users').authWithPassword(identity, password);
	}
	/** Re-pull the auth record (role changes, avatar). */
	async refresh() {
		try {
			await pb.collection('users').authRefresh();
		} catch {
			pb.authStore.clear();
		}
	}
	logout() {
		pb.authStore.clear();
	}
}

export const auth = new Auth();
