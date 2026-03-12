<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';
  interface Props {
    sessionId: string;
  }

  let { sessionId }: Props = $props();

  let terminalEl: HTMLDivElement | undefined = $state();
  let terminal: Terminal | undefined;
  let fitAddon: FitAddon | undefined;
  let ws: WebSocket | undefined;
  let resizeObserver: ResizeObserver | undefined;
  let reconnectTimeout: ReturnType<typeof setTimeout> | undefined;
  let colorSchemeQuery: MediaQueryList | undefined;
  let colorSchemeHandler: ((e: MediaQueryListEvent) => void) | undefined;

  const darkTheme = {
    foreground: '#e0e0e0',
    cursor: '#e0e0e0',
    selectionBackground: '#3a3a5c',
    red: '#f07178',
    green: '#c3e88d',
    yellow: '#ffcb6b',
    blue: '#82aaff',
    magenta: '#c792ea',
    cyan: '#89ddff',
    white: '#e0e0e0',
    brightBlack: '#545480',
    brightRed: '#f07178',
    brightGreen: '#c3e88d',
    brightYellow: '#ffcb6b',
    brightBlue: '#82aaff',
    brightMagenta: '#c792ea',
    brightCyan: '#89ddff',
    brightWhite: '#ffffff'
  };

  const lightTheme = {
    foreground: '#1e1e1e',
    cursor: '#1e1e1e',
    selectionBackground: '#add6ff',
    black: '#000000',
    red: '#cd3131',
    green: '#00bc00',
    yellow: '#949800',
    blue: '#0451a5',
    magenta: '#bc05bc',
    cyan: '#0598bc',
    white: '#e0e0e0',
    brightBlack: '#666666',
    brightRed: '#cd3131',
    brightGreen: '#14ce14',
    brightYellow: '#b5ba00',
    brightBlue: '#0451a5',
    brightMagenta: '#bc05bc',
    brightCyan: '#0598bc',
    brightWhite: '#a5a5a5'
  };

  function resolveTheme(isDark: boolean) {
    const bg = getComputedStyle(document.documentElement)
      .getPropertyValue('--color-surface')
      .trim();
    const base = isDark ? darkTheme : lightTheme;
    return { ...base, background: bg, black: bg };
  }

  function connect(term: Terminal) {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socket = new WebSocket(`${proto}//${window.location.host}/api/terminal`);
    socket.binaryType = 'arraybuffer';
    ws = socket;

    socket.addEventListener('open', () => {
      socket.send(
        JSON.stringify({
          type: 'init',
          sessionId,
          cols: term.cols,
          rows: term.rows
        })
      );
    });

    socket.addEventListener('message', (e) => {
      if (e.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(e.data));
      }
    });

    socket.addEventListener('close', (e) => {
      if (e.code === 1000 && e.reason === 'shell exited') {
        term.write('\r\n\x1b[90m[session ended]\x1b[0m\r\n');
      } else {
        // Connection lost — reset terminal and reconnect
        // The server will replay scrollback to restore the display
        reconnectTimeout = setTimeout(() => {
          term.reset();
          connect(term);
        }, 500);
      }
    });
  }

  onMount(() => {
    if (!terminalEl) return;

    colorSchemeQuery = window.matchMedia('(prefers-color-scheme: dark)');

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: '"JetBrainsMono NF", ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace',
      theme: resolveTheme(colorSchemeQuery.matches)
    });

    // Re-apply theme when color scheme changes
    colorSchemeHandler = (e) => {
      term.options.theme = resolveTheme(e.matches);
    };
    colorSchemeQuery.addEventListener('change', colorSchemeHandler);

    const fit = new FitAddon();
    fitAddon = fit;
    term.loadAddon(fit);
    term.open(terminalEl);

    // Initial fit after DOM layout settles
    requestAnimationFrame(() => fit.fit());

    // Connect WebSocket
    connect(term);

    // User input → WebSocket
    term.onData((data) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // Terminal resize → WebSocket
    term.onResize(({ cols, rows }) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols, rows }));
      }
    });

    // Re-fit when container resizes (e.g., SplitPane drag)
    resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(() => fit.fit());
    });
    resizeObserver.observe(terminalEl);

    terminal = term;
  });

  onDestroy(() => {
    if (reconnectTimeout) clearTimeout(reconnectTimeout);
    if (colorSchemeQuery && colorSchemeHandler) {
      colorSchemeQuery.removeEventListener('change', colorSchemeHandler);
    }
    resizeObserver?.disconnect();
    ws?.close();
    terminal?.dispose();
  });
</script>

<div class="flex h-full min-h-0 flex-col bg-surface">
  <div class="min-h-0 flex-1 p-2" bind:this={terminalEl}></div>
</div>
