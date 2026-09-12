import { pb } from './pb';

// Calls our custom Go endpoints. pb.send attaches the auth token and resolves
// relative to the SDK base URL (same origin).
async function post<T>(path: string, body: unknown): Promise<T> {
	return pb.send(path, { method: 'POST', body });
}
async function get<T>(path: string): Promise<T> {
	return pb.send(path, { method: 'GET' });
}
async function put<T>(path: string, body: unknown): Promise<T> {
	return pb.send(path, { method: 'PUT', body });
}
async function del<T>(path: string): Promise<T> {
	return pb.send(path, { method: 'DELETE' });
}

/** A pool's bound season, as the leagues API lists it. */
export interface PoolSeason {
	id: string;
	slug: string;
	name: string;
	shortName: string;
	status: string;
	competition?: { key: string; name: string; shortName: string };
}

/** A pool (the API still calls them leagues). */
export interface PoolSummary {
	id: string;
	name: string;
	inviteCode: string;
	role: string;
	private: boolean;
	members: number;
	/** Seasons the pool counts; empty for Global. */
	tournaments: PoolSeason[];
	/** live · upcoming · finished (all seasons over) · open (nothing bound). */
	status: 'live' | 'upcoming' | 'finished' | 'open';
}

export interface Person {
	userId: string;
	name: string;
	avatar?: string;
	/** Search results: '' | 'pending' (you asked) | 'incoming' (they asked) | 'accepted'. */
	state?: string;
}

export interface LeaderboardRow {
	userId: string;
	name: string;
	avatar?: string; // file name in the users.avatar field; empty/absent => none
	role?: string; // "admin" | "bot"; empty/absent => normal member
	total: number;
	tipsPoints: number;
	forecastPoints: number;
	predicted: number;
	exactScores: number;
	correctWinners: number;
	gdDeviation: number;
	forecast?: Record<string, number>;
}

export interface BotSummary {
	userId: string;
	name: string;
	avatar?: string;
	botKind?: string;
}

export interface ChatMessage {
	id: string;
	user: string; // sender user id
	text: string; // empty when deleted or a GIF
	gif?: string; // hosted GIF url (a message is text OR a gif)
	created: string; // RFC3339
	deleted?: boolean;
	// Moderation fields, returned only to app-admins for deleted messages:
	original?: string;
	originalGif?: string;
	deletedBy?: string;
	deletedAt?: string;
}

export interface GifResult {
	id: string;
	title: string;
	preview: string; // small url for the grid
	url: string; // full gif url to post
	width: number;
	height: number;
}

export interface ChatMember {
	userId: string;
	name: string;
	avatar?: string;
	role?: string;
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
}

// End-of-tournament feedback survey (v1). Enum values mirror the validation
// in internal/survey/survey.go.
export interface SurveyAnswers {
	v: number;
	enjoy: number; // 1–5
	playedOther: boolean;
	comparison?: 'better' | 'same' | 'worse';
	liked: string[];
	likedOther?: string;
	annoyed?: string;
	playAgain: 'definitely' | 'probably' | 'no';
	playSeason: 'definitely' | 'maybe' | 'no';
	competitions?: string[];
	competitionsOther?: string;
	fairPrice: 'donations' | 'fee' | 'nopay';
	comments?: string;
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

export const api = {
	joinPool: (code: string) =>
		post<{ id: string; name: string; already?: boolean }>(
			'/api/pools/join',
			{ code }
		),
	// Public — resolves an invite code to a league name for the /join page.
	invitePreview: (code: string) =>
		get<{ id: string; name: string }>(
			`/api/invite/${encodeURIComponent(code)}`
		),
	myPools: () => get<{ pools: PoolSummary[] }>('/api/pools/mine'),
	/** Standings of a league for one tournament (slug; default = the
	 *  server's current one). The response names the tournament used. */
	leaderboard: (id: string, tournament = '') =>
		get<{
			pool: { id: string; name: string };
			rows: LeaderboardRow[];
			/** Season the board was scored for; '' = the pool's seasons summed. */
			tournament?: string;
			tournaments?: PoolSeason[];
			status?: PoolSummary['status'];
			scoring?: Record<string, unknown>;
		}>(`/api/pools/${id}/leaderboard${tournament ? `?tournament=${encodeURIComponent(tournament)}` : ''}`),
	createPool: (name: string, tournaments: string[] = []) =>
		post<{ id: string; name: string; inviteCode: string }>('/api/pools/create', { name, tournaments }),
	setPoolSeasons: (id: string, tournaments: string[]) =>
		post<{ tournaments: PoolSeason[] }>(`/api/pools/${id}/tournaments`, { tournaments }),
	clonePool: (id: string, name: string, tournaments: string[]) =>
		post<{ id: string; name: string; inviteCode: string }>(`/api/pools/${id}/clone`, { name, tournaments }),
	// ---- friends: a mutual graph ----
	friends: () => get<{ friends: Person[]; incoming: Person[]; outgoing: Person[] }>('/api/friends'),
	searchPeople: (q: string) => get<{ users: Person[] }>(`/api/friends/search?q=${encodeURIComponent(q)}`),
	requestFriend: (userId: string) => post<{ state: string }>('/api/friends/request', { userId }),
	acceptFriend: (userId: string) => post<{ state: string }>('/api/friends/accept', { userId }),
	removeFriend: (userId: string) => post<{ state: string }>('/api/friends/remove', { userId }),
	friendsBoard: (tournament = '') =>
		get<{ tournament: string; rows: LeaderboardRow[] }>(
			`/api/friends/board${tournament ? `?tournament=${encodeURIComponent(tournament)}` : ''}`
		),
	// Owner-only league management.
	renamePool: (id: string, name: string) =>
		post<{ id: string; name: string }>(`/api/pools/${id}/rename`, { name }),
	regenerateCode: (id: string) =>
		post<{ inviteCode: string }>(`/api/pools/${id}/code/regenerate`, {}),
	setCodePrivacy: (id: string, isPrivate: boolean) =>
		post<{ private: boolean }>(`/api/pools/${id}/code/visibility`, {
			private: isPrivate
		}),
	removeMember: (id: string, userId: string) =>
		post<{ ok: boolean }>(`/api/pools/${id}/members/remove`, { userId }),
	// Owner-only: bot accounts not yet in the league, and adding one.
	availableBots: (id: string) =>
		get<{ bots: BotSummary[] }>(`/api/pools/${id}/bots`),
	addBot: (id: string, userId: string) =>
		post<{ ok: boolean; already?: boolean }>(`/api/pools/${id}/bots/add`, {
			userId
		}),
	// League chat (private leagues only).
	chatHistory: (poolId: string, before?: string) =>
		get<{ messages: ChatMessage[]; hasMore: boolean }>(
			`/api/pools/${poolId}/chat${before ? `?before=${encodeURIComponent(before)}` : ''}`
		),
	chatMembers: (poolId: string) =>
		get<{ members: ChatMember[] }>(`/api/pools/${poolId}/members`),
	chatPost: (poolId: string, body: { text?: string; gif?: string }) =>
		post<ChatMessage>(`/api/pools/${poolId}/chat`, body),
	chatGifSearch: (q: string, pos?: string) =>
		get<{ gifs: GifResult[]; next: string; configured: boolean }>(
			`/api/chat/gif/search?q=${encodeURIComponent(q)}${pos ? `&pos=${pos}` : ''}`
		),
	chatDelete: (poolId: string, msgId: string) =>
		del<ChatMessage>(`/api/pools/${poolId}/chat/${msgId}`),
	chatRestore: (poolId: string, msgId: string) =>
		post<ChatMessage>(`/api/pools/${poolId}/chat/${msgId}/restore`, {}),
	chatMarkRead: (poolId: string) =>
		post<{ ok: boolean }>(`/api/pools/${poolId}/chat/read`, {}),
	chatUnread: () => get<{ unread: Record<string, number> }>('/api/chat/unread'),

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
		post<{ status: string; logos: number }>(`/api/admin/tournaments/${id}/logos`, {}),
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

	// Feedback survey: own submission (for pre-filling) + submit/update.
	surveyStatus: () =>
		get<{ submitted: boolean; answers?: SurveyAnswers }>('/api/survey'),
	submitSurvey: (answers: SurveyAnswers) =>
		post<{ submitted: boolean }>('/api/survey', { answers })
};
