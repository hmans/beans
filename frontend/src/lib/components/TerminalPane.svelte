<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';
  import PaneHeader from './PaneHeader.svelte';

  interface Props {
    sessionId: string;
    onClose: () => void;
  }

  let { sessionId, onClose }: Props = $props();

  let terminalEl: HTMLDivElement | undefined = $state();
  let terminal: Terminal | undefined;
  let fitAddon: FitAddon | undefined;
  let ws: WebSocket | undefined;
  let resizeObserver: ResizeObserver | undefined;

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

    socket.addEventListener('close', () => {
      term.write('\r\n\x1b[90m[session ended]\x1b[0m\r\n');
    });
  }

  onMount(() => {
    if (!terminalEl) return;

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: '"JetBrainsMono NF", ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace',
      theme: {
        background: '#1a1a2e',
        foreground: '#e0e0e0',
        cursor: '#e0e0e0',
        selectionBackground: '#3a3a5c',
        black: '#1a1a2e',
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
      }
    });

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
    resizeObserver?.disconnect();
    ws?.close();
    terminal?.dispose();
  });
</script>

<div class="flex h-full flex-col border-l border-border bg-[#1a1a2e]">
  <PaneHeader title="Terminal" {onClose} />
  <div class="min-h-0 flex-1 px-1" bind:this={terminalEl}></div>
</div>
