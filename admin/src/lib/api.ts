import { pb } from './pb';

// The admin app's slice of the API: the admin endpoints plus the few reads
// the pages need. Types mirror frontend/src/lib/api.ts — keep the two in
// step when a payload changes.

async function post<T>(path: string, body: unknown): Promise<T> {
	return pb.send<T>(path, { method: 'POST', body });
}
async function get<T>(path: string): Promise<T> {
	return pb.send<T>(path, { method: 'GET' });
}
async function put<T>(path: string, body: unknown): Promise<T> {
	return pb.send<T>(path, { method: 'PUT', body });
}
async function del<T>(path: string): Promise<T> {
	return pb.send<T>(path, { method: 'DELETE' });
}

/** The pools the signed-in admin is in (the dev harness picks one for bots). */
export interface PoolSummary {
	id: string;
	name: string;
	mode?: string;
	inviteCode?: string; // 'GLOBAL' marks the everyone pool
	status?: string; // open | live | upcoming | finished
}

export type AnnounceLevel = 'info' | 'success' | 'warn';

export interface Announcement {
	id: string;
	title: string;
	body: string;
	level: AnnounceLevel;
	active: boolean;
	highPriority: boolean; // high-urgency push when broadcast
	persistent: boolean; // can't be dismissed — only collapsed
	ctaText: string; // button text ('' = the default "Open Matchowl")
	ctaUrl: string; // button link (URL or in-app path)
	notifiedAt: string; // RFC3339, empty if never broadcast
	created: string;
}

export interface AnnouncePayload {
	title?: string;
	body?: string;
	level?: AnnounceLevel;
	active?: boolean;
	highPriority?: boolean;
	persistent?: boolean;
	ctaText?: string;
	ctaUrl?: string;
}

// ---- Mailings: one targeted email to an audience ----

/** Recipient filter; every set criterion narrows. */
export interface Audience {
	roles?: string[]; // member | admin | owner
	playing?: string; // tournament id
	pool?: string; // pool id
	joinedAfter?: string; // YYYY-MM-DD
	joinedBefore?: string;
	emails?: string[]; // only these
	exclude?: string[]; // never these
}
export interface Mailing {
	id: string;
	subject: string;
	body: string; // Markdown; {{name}} = the recipient's first name
	ctaText: string;
	ctaUrl: string;
	audience: Audience;
	status: 'draft' | 'sent';
	sentAt: string;
	recipients: number;
	result: { considered: number; sent: number; failed: number; skipped: number } | null;
	created: string;
	updated: string;
}
export interface MailingPayload {
	subject?: string;
	body?: string;
	ctaText?: string;
	ctaUrl?: string;
	audience?: Audience;
}

export type NotifyChannel = 'email' | 'push';

// Global notification delivery policy (app_meta notify_config). `channels` are
// the master per-channel kill switches; `disabled[event][channel] === true`
// suppresses one event's channel even when the master switch is on.
export interface NotifyPolicy {
	channels: Record<NotifyChannel, boolean>;
	disabled: Record<string, Partial<Record<NotifyChannel, boolean>>>;
}

// PUT body — both sections optional so callers can patch one at a time.
export interface NotifyPolicyPayload {
	channels?: Partial<Record<NotifyChannel, boolean>>;
	disabled?: Record<string, Partial<Record<NotifyChannel, boolean>>>;
}

export interface SyncLastRun {
	at: string; // RFC3339
	source: string;
	updated: number;
	ok: boolean;
	error?: string;
}

export interface SyncStatus {
	/** Active tournaments with a resolved results source. */
	sources: { tournament: string; source: string }[]; // source: 'api-football' | 'openfootball'
	/** Active tournaments with no usable source, and why. */
	skipped: { tournament: string; reason: string }[];
	/** Set when `sources` is empty: why nothing syncs. */
	reason?: string;
	autoSync: boolean;
	cron: string;
	/** Keyed by tournament slug. */
	lastRun: Record<string, SyncLastRun> | null;
	account?: {
		subscription?: { plan?: string; active?: boolean; end?: string };
		requests?: { current?: number; limit_day?: number };
	};
	accountError?: string;
}

export interface GifStatus {
	configured: boolean;
	gifsSent: number; // GIF messages ever posted (deleted ones included)
	health?: { ok: boolean; latencyMs?: number; error?: string };
}

/** A one-time registration link (closed test), as the admin list shows it. */
export interface SignupLink {
	id: string;
	token: string;
	label: string;
	status: 'open' | 'used' | 'expired';
	created: string; // RFC3339
	expiresAt: string; // '' = never
	usedAt: string; // '' = not yet
	createdBy?: { id: string; name: string };
	usedBy?: { id: string; name: string; email: string };
	pools: { id: string; name: string }[]; // joined once the account is verified
}

export interface OwnerStats {
	users: number; // real users (bots excluded)
	usersLast24h: number;
	activeUsers: number; // >=3 tips or a complete forecast
	pools: number; // user-created (Global excluded)
	activeLeagues: number; // >1 member and some tips
	pushEnabled: number;
	notifyDisabled: number; // opted out of >=1 notification
	emailReachable: number; // verified AND master email switch on
	emailOptedOut: number; // master email switch off
	unverified: number; // never verified their address
}

// Deploy-time settings the frontend needs at runtime (from env vars).
export interface AppConfig {
	kofiUrl: string; // empty = no Ko-Fi configured, hide the support card
	contactEmail: string;
	operatorName: string; // who runs Matchowl ('' until configured)
	operatorLocation: string; // town + country, never a street
	version: string; // build version, e.g. "1.0.0-alpha.1" ("dev" locally)
	registrationOpen: boolean; // false = private testing, no new accounts
	appUrl: string; // the player app's public origin ('' when unset)
	adminUrl: string; // the admin app's origin ('' when none)
}

// ---- Admin: tournaments, catalog import ----

export type TournamentStatus = 'draft' | 'upcoming' | 'active' | 'finished' | 'archived';

/** Admin view of a tournament (drafts included, plus seed counts). */
export interface AdminTournament {
	id: string;
	slug: string;
	name: string;
	shortName: string;
	status: TournamentStatus;
	startsAt: string;
	endsAt: string;
	structure: Record<string, unknown>;
	forecastSpec: Record<string, unknown>;
	sync: Record<string, unknown> | null;
	extIdPrefix: string;
	scoringConfig: string;
	competition: string; // competition id or ''
	teams: number;
	matches: number;
	players: number;
}

/** Create/update body — every field optional; omitted = unchanged. */
export interface AdminTournamentPayload {
	slug?: string;
	name?: string;
	shortName?: string;
	status?: TournamentStatus;
	startsAt?: string;
	endsAt?: string;
	structure?: unknown;
	sync?: unknown;
	forecastSpec?: unknown;
	extIdPrefix?: string;
	scoringConfig?: string;
	competition?: string;
}

export interface AdminCompetition {
	id: string;
	key: string;
	name: string;
	shortName: string;
	country: string;
	teamKind: 'national' | 'club';
	logo: string;
	description: string;
	apiFootballLeague: number;
}

export interface AdminCompetitionPayload {
	key?: string;
	name?: string;
	shortName?: string;
	country?: string;
	teamKind?: 'national' | 'club';
	description?: string;
	apiFootballLeague?: number;
}

/** API-Football catalog entry. */
export interface FootballLeague {
	id: number;
	name: string;
	type: 'League' | 'Cup' | string;
	logo: string;
	country: string;
	flag: string;
	seasons: { year: number; start: string; end: string; current: boolean }[];
}

/** Import proposal derived from a league season (editable before import). */
export interface ImportProposal {
	poolId: number;
	leagueName: string;
	leagueType: string;
	leagueLogo: string;
	country: string;
	season: number;
	teamKind: 'national' | 'club';
	slug: string;
	name: string;
	shortName: string;
	extIdPrefix: string;
	startsAt: string;
	endsAt: string;
	structure: Record<string, unknown>;
	sync: Record<string, unknown>;
	forecastSpec: Record<string, unknown>;
	shape: string;
	teams: {
		id: number;
		name: string;
		code: string;
		country: string;
		national: boolean;
		logo: string;
		group?: string;
		iso2?: string;
	}[];
	groups: { letter: string; teams: string[] }[];
	rounds: { label: string; stage: string; matches: number; first: string }[];
	fixtures: number;
	warnings: string[] | null;
}

// ---- Admin: group editor + teams registry ----

export interface GroupTeam {
	id: string;
	name: string;
	fifaCode: string;
	iso2: string;
	logo: string;
}
export interface GroupsView {
	groups: { id: string; letter: string; teams: GroupTeam[] }[];
	unassigned: GroupTeam[];
	structure: { groupSize: number; gamesPerTeam: number; hasGroups: boolean } | null;
	groupMatches: number;
}
export interface RegistryEntry {
	teamId: string;
	tournament: string;
	slug: string;
	season: string;
	status: string;
	group: string;
	fifaCode: string;
	iso2: string;
	logo: string;
	name: string;
}
/** One team across every season it plays in (grouped by provider id, club key or name). */
export interface RegistryTeam {
	key: string;
	name: string;
	fifaCode: string;
	iso2: string;
	clubKey: string;
	provider: string;
	providerId: number;
	/** "<teamId>/<file>" of the first row with a crest. */
	logo: string;
	/** Fields that differ across the rows. */
	mixed: string[];
	entries: RegistryEntry[];
}

export const api = {
	myPools: () => get<{ pools: PoolSummary[] }>('/api/pools/mine'),

	// Global notification policy: read (any signed-in user, so settings can show
	// force-disabled toggles) + owner/admin write.
	notifyPolicy: () => get<NotifyPolicy>('/api/notify/policy'),
	updateNotifyPolicy: (p: NotifyPolicyPayload) =>
		put<NotifyPolicy>('/api/admin/notify/policy', p),

	// Owner-only app stats dashboard.
	ownerStats: () => get<OwnerStats>('/api/stats/owner'),


	// Admin: tournament management + API-Football catalog import.
	adminTournaments: () => get<{ tournaments: AdminTournament[] }>('/api/admin/tournaments'),
	adminTournamentCreate: (body: AdminTournamentPayload) =>
		post<AdminTournament>('/api/admin/tournaments', body),
	adminTournamentUpdate: (id: string, body: AdminTournamentPayload) =>
		post<AdminTournament>(`/api/admin/tournaments/${id}`, body),
	adminTournamentDelete: (id: string) => del<{ ok: boolean }>(`/api/admin/tournaments/${id}`),
	adminTournamentSeed: (id: string, teams: unknown, fixtures: unknown) =>
		post<{ status: string; teams: number; matches: number }>(
			`/api/admin/tournaments/${id}/seed`,
			{ teams, fixtures }
		),
	adminTournamentLogos: (id: string) =>
		post<{ status: string; logos: number; linked: number }>(`/api/admin/tournaments/${id}/logos`, {}),
	adminCompetitions: () => get<{ competitions: AdminCompetition[] }>('/api/admin/competitions'),
	adminCompetitionUpdate: (id: string, body: AdminCompetitionPayload) =>
		post<AdminCompetition>(`/api/admin/competitions/${id}`, body),
	adminCompetitionLogo: (id: string) =>
		post<AdminCompetition>(`/api/admin/competitions/${id}/logo`, {}),
	footballLeagues: (search: string) =>
		get<{ leagues: FootballLeague[] }>(
			`/api/admin/football/leagues?search=${encodeURIComponent(search)}`
		),
	footballPreview: (league: number, season: number) =>
		get<ImportProposal>(`/api/admin/football/preview?league=${league}&season=${season}`),
	tournamentImport: (
		proposal: ImportProposal,
		extra: { competition?: string; status?: TournamentStatus } = {}
	) =>
		post<{ id: string; slug: string; teams: number; matches: number }>(
			'/api/admin/tournaments/import',
			{ ...proposal, ...extra }
		),

	// Owner-only results-sync dashboard: status + manual trigger.
	syncStatus: () => get<SyncStatus>('/api/admin/sync/status'),
	syncRun: () =>
		post<{ status?: string; updated?: number; error?: string; lastRun: SyncLastRun | null }>(
			'/api/admin/sync/run',
			{}
		),

	// Admin-only GIF-API dashboard: KLIPY health + GIFs-sent count.
	gifStatus: () => get<GifStatus>('/api/admin/gif/status'),

	// One-time registration links for the closed test (admin), plus the
	// anonymous check the register page runs on the token it was opened with.
	signupLinks: () => get<{ links: SignupLink[] }>('/api/admin/signup-links'),
	createSignupLink: (label: string, expiresInDays = 0, pools: string[] = []) =>
		post<SignupLink>('/api/admin/signup-links', { label, expiresInDays, pools }),
	revokeSignupLink: (id: string) =>
		del<{ ok: boolean }>(`/api/admin/signup-links/${encodeURIComponent(id)}`),
	checkSignupLink: (token: string) =>
		get<{ ok: boolean; label?: string; code?: string }>(
			`/api/signup-links/${encodeURIComponent(token)}`
		),

	// Announcements: active list (any signed-in user, for the banner) + the
	// owner/admin-only management endpoints.
	activeAnnouncements: () =>
		get<{ announcements: Announcement[] }>('/api/announce/active'),
	allAnnouncements: () =>
		get<{ announcements: Announcement[] }>('/api/admin/announce'),
	createAnnouncement: (p: AnnouncePayload) =>
		post<Announcement>('/api/admin/announce', p),
	updateAnnouncement: (id: string, p: AnnouncePayload) =>
		post<Announcement>(`/api/admin/announce/${id}`, p),
	deleteAnnouncement: (id: string) =>
		del<{ ok: boolean }>(`/api/admin/announce/${id}`),
	sendAnnouncement: (id: string) =>
		post<{
			announcement: Announcement;
			result: { considered: number; sent: number; failed: number; skipped: number };
		}>(`/api/admin/announce/${id}/send`, {}),

	// Public deploy-time config (Ko-Fi / contact links).
	appConfig: () => get<AppConfig>('/api/appconfig'),

	// Group editor + teams registry.
	adminGroups: (tournamentId: string) => get<GroupsView>(`/api/admin/tournaments/${tournamentId}/groups`),
	adminSaveGroups: (tournamentId: string, groups: { letter: string; teams: string[] }[]) =>
		put<GroupsView>(`/api/admin/tournaments/${tournamentId}/groups`, { groups }),
	adminTeams: () => get<{ teams: RegistryTeam[] }>('/api/admin/teams'),
	adminTeamUpdate: (id: string, body: { name?: string; fifaCode?: string; iso2?: string; applyToAll?: boolean }) =>
		post<{ updated: number }>(`/api/admin/teams/${id}`, body),
	// Mailings.
	mailings: () => get<{ mailings: Mailing[] }>('/api/admin/mailings'),
	createMailing: (p: MailingPayload) => post<Mailing>('/api/admin/mailings', p),
	updateMailing: (id: string, p: MailingPayload) => post<Mailing>(`/api/admin/mailings/${id}`, p),
	deleteMailing: (id: string) => del<{ ok: boolean }>(`/api/admin/mailings/${id}`),
	mailingAudience: (audience: Audience) =>
		post<{ count: number; recipients: { id: string; name: string; email: string }[] }>('/api/admin/mailings/audience', { audience }),
	/** The mail as a recipient sees it (event: 'mailing' or 'announcement'). */
	mailRender: (p: { subject: string; body: string; ctaText: string; ctaUrl: string; event?: 'mailing' | 'announcement' }) =>
		post<{ subject: string; html: string; text: string }>('/api/admin/mailings/render', p),
	mailingTest: (id: string) => post<{ to: string; provider: string }>(`/api/admin/mailings/${id}/test`, {}),
	mailingSend: (id: string) =>
		post<{ mailing: Mailing; recipients: number; result: { considered: number; sent: number; failed: number; skipped: number } }>(`/api/admin/mailings/${id}/send`, {}),
};
