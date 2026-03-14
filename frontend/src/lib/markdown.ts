import { Marked, type MarkedExtension } from 'marked';
import DOMPurify from 'dompurify';
import { browser } from '$app/environment';
import { createHighlighterCore, type HighlighterCore } from 'shiki/core';
import { createOnigurumaEngine } from 'shiki/engine/oniguruma';
import githubDark from 'shiki/themes/github-dark.mjs';
import githubLight from 'shiki/themes/github-light.mjs';

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
import langRuby from 'shiki/langs/ruby.mjs';
import langPython from 'shiki/langs/python.mjs';
import langRust from 'shiki/langs/rust.mjs';
import langSql from 'shiki/langs/sql.mjs';
import langHtml from 'shiki/langs/html.mjs';
import langCss from 'shiki/langs/css.mjs';
import langSvelte from 'shiki/langs/svelte.mjs';
import langJsx from 'shiki/langs/jsx.mjs';
import langTsx from 'shiki/langs/tsx.mjs';
import langToml from 'shiki/langs/toml.mjs';
import langDockerfile from 'shiki/langs/dockerfile.mjs';
import langC from 'shiki/langs/c.mjs';
import langCpp from 'shiki/langs/cpp.mjs';
import langJava from 'shiki/langs/java.mjs';
import langPhp from 'shiki/langs/php.mjs';
import langSwift from 'shiki/langs/swift.mjs';
import langKotlin from 'shiki/langs/kotlin.mjs';
import langCsharp from 'shiki/langs/csharp.mjs';
import langLua from 'shiki/langs/lua.mjs';
import langElixir from 'shiki/langs/elixir.mjs';

const BUNDLED_LANGS = [
  langJavascript,
  langTypescript,
  langJsx,
  langTsx,
  langGo,
  langBash,
  langJson,
  langYaml,
  langMarkdown,
  langGraphql,
  langDiff,
  langRuby,
  langPython,
  langRust,
  langSql,
  langHtml,
  langCss,
  langSvelte,
  langToml,
  langDockerfile,
  langC,
  langCpp,
  langJava,
  langPhp,
  langSwift,
  langKotlin,
  langCsharp,
  langLua,
  langElixir
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
  'diff',
  'ruby',
  'rb',
  'python',
  'py',
  'rust',
  'rs',
  'sql',
  'html',
  'css',
  'svelte',
  'jsx',
  'tsx',
  'toml',
  'dockerfile',
  'docker',
  'c',
  'cpp',
  'c++',
  'java',
  'php',
  'swift',
  'kotlin',
  'kt',
  'csharp',
  'c#',
  'cs',
  'lua',
  'elixir',
  'ex'
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
  gql: 'graphql',
  rb: 'ruby',
  py: 'python',
  rs: 'rust',
  docker: 'dockerfile',
  'c++': 'cpp',
  kt: 'kotlin',
  'c#': 'csharp',
  cs: 'csharp',
  ex: 'elixir'
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
    themes: [githubDark, githubLight],
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
              themes: {
                light: 'github-light',
                dark: 'github-dark'
              },
              defaultColor: false,
              cssVariablePrefix: '--shiki-'
            });
          } catch {
            // Fall through to plain rendering
          }
        }

        // Fallback for unsupported languages
        const escaped = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        return `<pre class="shiki"><code>${escaped}</code></pre>`;
      }
    }
  };
}

/**
 * Marked extension that auto-links bean IDs (e.g. beans-s1m0) in inline text.
 * Renders as <a data-bean-id="beans-xxxx"> so click handlers can navigate.
 */
function beanLinkExtension(): MarkedExtension {
  return {
    extensions: [
      {
        name: 'beanLink',
        level: 'inline',
        start(src: string) {
          return src.match(/beans-/)?.index;
        },
        tokenizer(src: string) {
          const match = src.match(/^beans-[a-z0-9]{4}\b/);
          if (match) {
            return {
              type: 'beanLink',
              raw: match[0],
              beanId: match[0]
            };
          }
          return undefined;
        },
        renderer(token) {
          const beanId = (token as Record<string, unknown>).beanId as string;
          return `<a data-bean-id="${beanId}" class="bean-link">${beanId}</a>`;
        }
      }
    ]
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
  md.use(beanLinkExtension());
  md.use({
    renderer: {
      link({ href, title, tokens }) {
        const text = this.parser.parseInline(tokens);
        const titleAttr = title ? ` title="${title}"` : '';
        return `<a href="${href}"${titleAttr} target="_blank" rel="noopener noreferrer">${text}</a>`;
      }
    }
  });

  const hl = await getHighlighter();
  if (hl) {
    md.use(shikiExtension(hl));
  } else {
    md.use(plainCodeExtension());
  }

  const html = md.parse(content) as string;

  if (browser) {
    return DOMPurify.sanitize(html, {
      ADD_TAGS: ['input'],
      ADD_ATTR: ['data-bean-id', 'target', 'rel', 'checked', 'disabled', 'type']
    });
  }

  return html;
}

/**
 * Plain code block extension for SSR (no syntax highlighting)
 */
function plainCodeExtension(): MarkedExtension {
  return {
    renderer: {
      code({ text }) {
        const escaped = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        return `<pre class="shiki"><code>${escaped}</code></pre>`;
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

