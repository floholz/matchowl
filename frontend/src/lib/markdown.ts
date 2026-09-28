// The tiny Markdown subset admin copy uses (announcements, mailings),
// rendered to safe HTML: paragraphs, headings, bullet lists, bold, italic,
// links and bare URLs. Mirrors internal/markdown in Go (the mail side);
// keep the two in step.

const esc = (s: string) =>
	s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
const safe = (u: string) => /^(https?:\/\/|mailto:|\/)/i.test(u);

function inline(s: string): string {
	s = esc(s);
	s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_m, t: string, u: string) =>
		safe(u) ? `<a href="${u}">${t}</a>` : t
	);
	s = s.replace(/(^|[\s(])(https?:\/\/[^\s<)]+)/g, (m, pre: string, u: string) =>
		m.includes('href=') ? m : `${pre}<a href="${u}">${u}</a>`
	);
	s = s.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
	s = s.replace(/(^|[^*\w])[*_]([^*_\n]+?)[*_]/g, '$1<em>$2</em>');
	return s;
}

export function renderMarkdown(src: string): string {
	const out: string[] = [];
	for (const raw of src.replace(/\r\n/g, '\n').trim().split('\n\n')) {
		const block = raw.trim();
		if (!block) continue;
		const lines = block.split('\n');
		if (block.startsWith('## ')) {
			out.push(`<h2>${inline(lines[0].slice(3))}</h2>`);
			if (lines.length > 1) out.push(`<p>${lines.slice(1).map((l) => inline(l.trim())).join('<br>')}</p>`);
		} else if (block.startsWith('# ')) {
			out.push(`<h1>${inline(lines[0].slice(2))}</h1>`);
			if (lines.length > 1) out.push(`<p>${lines.slice(1).map((l) => inline(l.trim())).join('<br>')}</p>`);
		} else if (lines.every((l) => /^\s*[-*] /.test(l))) {
			out.push(`<ul>${lines.map((l) => `<li>${inline(l.trim().slice(2))}</li>`).join('')}</ul>`);
		} else {
			out.push(`<p>${lines.map((l) => inline(l.trim())).join('<br>')}</p>`);
		}
	}
	return out.join('');
}

/** Markdown down to plain text (one-line previews). */
export function plainMarkdown(src: string): string {
	return src
		.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, '$1')
		.replace(/\*\*(.+?)\*\*/g, '$1')
		.replace(/(^|[^*\w])[*_]([^*_\n]+?)[*_]/g, '$1$2')
		.replace(/^#+\s*/gm, '')
		.replace(/^\s*[-*]\s+/gm, '')
		.trim();
}
