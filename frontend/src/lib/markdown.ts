import { Marked, type MarkedExtension } from 'marked';
import { browser } from '$app/environment';
import { createHighlighterCore, type HighlighterCore } from 'shiki/core';
import { createOnigurumaEngine } from 'shiki/engine/oniguruma';
import githubDark from 'shiki/themes/github-dark.mjs';

// Import only essential languages to keep bundle small
import langJavascript from 'shiki/langs/javascript.mjs';
import langTypescript from 'shiki/langs/typescript.mjs';
import langGo from 'shiki/langs/go.mjs';
import langBash from 'shiki/langs/bash.mjs';
import langJson from 'shiki/langs/json.mjs';
import langYaml from 'shiki/langs/yaml.mjs';
import langMarkdown from 'shiki/langs/markdown.mjs';
import langGraphql from 'shiki/langs/graphql.mjs';
import langDiff from 'shiki/langs/diff.mjs';

const BUNDLED_LANGS = [
	langJavascript,
	langTypescript,
	langGo,
	langBash,
	langJson,
	langYaml,
	langMarkdown,
	langGraphql,
	langDiff
];

// Languages we have bundled (including aliases)
const SUPPORTED_LANGS = new Set([
	'javascript',
	'js',
	'typescript',
	'ts',
	'go',
	'bash',
	'sh',
	'shell',
	'zsh',
	'json',
	'yaml',
	'yml',
	'markdown',
	'md',
	'graphql',
	'gql',
	'diff'
]);

// Common language aliases
const LANG_ALIASES: Record<string, string> = {
	js: 'javascript',
	ts: 'typescript',
	sh: 'bash',
	zsh: 'bash',
	shell: 'bash',
	yml: 'yaml',
	md: 'markdown',
	gql: 'graphql'
};

let highlighter: HighlighterCore | null = null;
let highlighterPromise: Promise<HighlighterCore> | null = null;

/**
 * Initialize the shiki highlighter with bundled languages only.
 * Only works in browser - returns null during SSR.
 */
async function getHighlighter(): Promise<HighlighterCore | null> {
	if (!browser) return null;

	if (highlighter) return highlighter;
	if (highlighterPromise) return highlighterPromise;

	highlighterPromise = createHighlighterCore({
		engine: createOnigurumaEngine(import('shiki/wasm')),
		themes: [githubDark],
		langs: BUNDLED_LANGS
	});

	highlighter = await highlighterPromise;
	return highlighter;
}

/**
 * Custom marked extension for shiki syntax highlighting
 */
function shikiExtension(hl: HighlighterCore): MarkedExtension {
	return {
		renderer: {
			code({ text, lang }) {
				const rawLang = (lang || '').toLowerCase();
				const language = LANG_ALIASES[rawLang] || rawLang;

				// Only highlight supported languages
				if (SUPPORTED_LANGS.has(rawLang) || SUPPORTED_LANGS.has(language)) {
					try {
						return hl.codeToHtml(text, {
							lang: language,
							theme: 'github-dark'
						});
					} catch {
						// Fall through to plain rendering
					}
				}

				// Fallback for unsupported languages
				const escaped = text
					.replace(/&/g, '&amp;')
					.replace(/</g, '&lt;')
					.replace(/>/g, '&gt;');
				return `<pre class="shiki" style="background-color:#24292e;color:#e1e4e8"><code>${escaped}</code></pre>`;
			}
		}
	};
}

/**
 * Render markdown to HTML with syntax highlighting.
 * Falls back to plain code blocks during SSR.
 */
export async function renderMarkdown(content: string): Promise<string> {
	if (!content) return '';

	const md = new Marked();
	md.use({ gfm: true, breaks: true });

	const hl = await getHighlighter();
	if (hl) {
		md.use(shikiExtension(hl));
	} else {
		md.use(plainCodeExtension());
	}

	return md.parse(content) as string;
}

/**
 * Plain code block extension for SSR (no syntax highlighting)
 */
function plainCodeExtension(): MarkedExtension {
	return {
		renderer: {
			code({ text }) {
				const escaped = text
					.replace(/&/g, '&amp;')
					.replace(/</g, '&lt;')
					.replace(/>/g, '&gt;');
				return `<pre class="shiki" style="background-color:#24292e;color:#e1e4e8"><code>${escaped}</code></pre>`;
			}
		}
	};
}

/**
 * Pre-initialize the highlighter (call on app start for faster first render)
 */
export function preloadHighlighter(): void {
	getHighlighter().catch(console.error);
}
