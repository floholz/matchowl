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

/** How a pool plays its season: one points table, or matchday duels. */
export type PoolMode = 'classic' | 'h2h';

/** The mode block every pool payload carries. */
export interface PoolModeInfo {
	mode: PoolMode;
	/** Head-to-head: matches per matchday that count double (1 or 2). */
	saveCalls: number;
	/** When season / mode / save calls freeze (RFC3339): the first kick-off
	 *  of the pool's first round. '' when the pool binds no season. */
	lockAt: string;
	locked: boolean;
}

/** A pool (the API still calls them leagues). */
export interface PoolSummary extends PoolModeInfo {
	id: string;
	name: string;
	inviteCode: string;
	role: string;
	private: boolean;
	members: number;
	/** The one season the pool plays (a list for history's sake); empty for Global. */
	tournaments: PoolSeason[];
	/** live · upcoming · finished (all seasons over) · open (nothing bound). */
	status: 'live' | 'upcoming' | 'finished' | 'open';
	/** The chat still takes messages (a finished pool's chat stays open a while). */
	chatOpen: boolean;
	/** When the chat closes (RFC3339), '' while any season is on. */
	chatUntil: string;
}

/** What a pool plays and how; the create / settings / clone payload. */
export interface PoolSettings {
	/** Season slug. */
	tournament: string;
	mode: PoolMode;
	saveCalls: number;
}

/** Head-to-head: a member as the duel views name them; null = the Ghost. */
export interface H2HPerson {
	userId: string;
	name: string;
	avatar: string;
	role: string;
}
export interface H2HPair {
	a: H2HPerson;
	b: H2HPerson | null;
	scoreA: number;
	scoreB: number;
	/** 3 / 1 / 0 each. */
	ptsA: number;
	ptsB: number;
}
export interface H2HRound {
	key: string;
	label: string;
	num: number;
	ordinal: number;
	status: 'open' | 'closed';
	firstKickoff: string;
	closesAt: string;
	pairs: H2HPair[];
	/** Matches in the round, and how many of them count so far. */
	matches: number;
	counted: number;
	ghost: number;
}
export interface H2HRoundDetail extends H2HRound {
	matchIds: string[];
	countedIds: string[];
	/** user → match → tip points behind the scores. */
	breakdown: Record<string, Record<string, number>>;
}
export interface H2HTableRow {
	userId: string;
	name: string;
	avatar: string;
	role: string;
	played: number;
	won: number;
	drawn: number;
	lost: number;
	points: number;
	tipsPoints: number;
	scoreFor: number;
	scoreAgainst: number;
}
export interface H2HOverview {
	table: H2HTableRow[];
	rounds: H2HRound[];
	/** Key of the round to show first: the open one, else the last closed. */
	current: string;
	/** The round that opens next, with the pairings the roster would get. */
	next: {
		key: string;
		label: string;
		num: number;
		firstKickoff: string;
		closesAt: string;
		matches: number;
		pairs: { a: H2HPerson; b: H2HPerson | null }[];
	} | null;
	firstRound: { key: string; label: string; firstKickoff: string };
	saveCalls: number;
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
	// Public — resolves an invite code to a pool name (and whether the pool
	// is finished) for the /join page.
	invitePreview: (code: string) =>
		get<{ id: string; name: string; finished: boolean }>(
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
			chatOpen?: boolean;
			chatUntil?: string;
			scoring?: Record<string, unknown>;
		} & Partial<PoolModeInfo>>(`/api/pools/${id}/leaderboard${tournament ? `?tournament=${encodeURIComponent(tournament)}` : ''}`),
	/** Mode and save calls a pool for the season gets unless the creator says otherwise. */
	poolDefaults: (tournament: string) =>
		get<{ mode: PoolMode; saveCalls: number; matchesPerRound: number }>(
			`/api/pools/defaults?tournament=${encodeURIComponent(tournament)}`
		),
	createPool: (name: string, settings: PoolSettings) =>
		post<{ id: string; name: string; inviteCode: string; mode: PoolMode }>('/api/pools/create', { name, ...settings }),
	// ---- head-to-head pools ----
	h2h: (poolId: string) => get<H2HOverview>(`/api/pools/${poolId}/h2h`),
	h2hRound: (poolId: string, key: string) =>
		get<H2HRoundDetail>(`/api/pools/${poolId}/h2h/round?key=${encodeURIComponent(key)}`),
	/** Owner: the season the pool plays and how — until its first round kicks off. */
	setPoolSettings: (id: string, settings: Partial<PoolSettings>) =>
		post<{ tournaments: PoolSeason[] } & PoolModeInfo>(`/api/pools/${id}/settings`, settings),
	clonePool: (id: string, name: string, settings: Partial<PoolSettings>) =>
		post<{ id: string; name: string; inviteCode: string; mode: PoolMode }>(`/api/pools/${id}/clone`, { name, ...settings }),
	// ---- pool invites: members bring friends in without a code ----
	invitable: (poolId: string) =>
		get<{ friends: (Person & { invited: boolean })[] }>(`/api/pools/${poolId}/invitable`),
	invite: (poolId: string, userId: string) =>
		post<{ state: string }>(`/api/pools/${poolId}/invite`, { userId }),
	poolInvites: () =>
		get<{
			invites: { id: string; pool: { id: string; name: string; members: number; tournaments: PoolSeason[] }; from: string }[];
		}>('/api/pools/invites'),
	acceptInvite: (id: string) => post<{ id: string; name: string }>(`/api/pools/invites/${id}/accept`, {}),
	declineInvite: (id: string) => post<{ ok: boolean }>(`/api/pools/invites/${id}/decline`, {}),
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

	// One-time registration links for the closed test (admin), plus the
	// anonymous check the register page runs on the token it was opened with.
	signupLinks: () => get<{ links: SignupLink[] }>('/api/admin/signup-links'),
	createSignupLink: (label: string, expiresInDays = 0) =>
		post<SignupLink>('/api/admin/signup-links', { label, expiresInDays }),
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

	// Feedback survey: own submission (for pre-filling) + submit/update.
	surveyStatus: () =>
		get<{ submitted: boolean; answers?: SurveyAnswers }>('/api/survey'),
	submitSurvey: (answers: SurveyAnswers) =>
		post<{ submitted: boolean }>('/api/survey', { answers })
};
