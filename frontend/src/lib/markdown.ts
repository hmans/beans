import { Marked, type MarkedExtension } from 'marked';
import { createHighlighter, type Highlighter } from 'shiki';

let highlighter: Highlighter | null = null;
let highlighterPromise: Promise<Highlighter> | null = null;

/**
 * Initialize the shiki highlighter (lazy, singleton)
 */
async function getHighlighter(): Promise<Highlighter> {
	if (highlighter) return highlighter;
	if (highlighterPromise) return highlighterPromise;

	highlighterPromise = createHighlighter({
		themes: ['github-dark'],
		langs: [
			'javascript',
			'typescript',
			'go',
			'bash',
			'shell',
			'json',
			'yaml',
			'markdown',
			'html',
			'css',
			'sql',
			'graphql',
			'python',
			'rust',
			'diff'
		]
	});

	highlighter = await highlighterPromise;
	return highlighter;
}

/**
 * Custom marked extension for shiki syntax highlighting
 */
function shikiExtension(hl: Highlighter): MarkedExtension {
	return {
		renderer: {
			code({ text, lang }) {
				const language = lang || 'text';
				try {
					// Check if language is loaded
					const loadedLangs = hl.getLoadedLanguages();
					if (loadedLangs.includes(language as typeof loadedLangs[number])) {
						return hl.codeToHtml(text, {
							lang: language,
							theme: 'github-dark'
						});
					}
				} catch {
					// Fall through to default rendering
				}
				// Fallback for unknown languages
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
 * Render markdown to HTML with syntax highlighting
 */
export async function renderMarkdown(content: string): Promise<string> {
	if (!content) return '';

	const hl = await getHighlighter();

	// Create a new marked instance with our extensions
	const md = new Marked();
	md.use({ gfm: true, breaks: true });
	md.use(shikiExtension(hl));

	return md.parse(content) as string;
}

/**
 * Pre-initialize the highlighter (call on app start for faster first render)
 */
export function preloadHighlighter(): void {
	getHighlighter().catch(console.error);
}
