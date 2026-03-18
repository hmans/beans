import { test as base } from '@playwright/test';
import { type ChildProcess, execFileSync, spawn } from 'node:child_process';
import { cpSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { connect } from 'node:net';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { BacklogPage } from './pages/backlog-page';
import { BoardPage } from './pages/board-page';

const PROJECT_ROOT = join(import.meta.dirname, '../..');
const BASE_PORT = 22900;

const GIT_ENV = {
  ...process.env,
  GIT_AUTHOR_NAME: 'test',
  GIT_AUTHOR_EMAIL: 'test@test',
  GIT_COMMITTER_NAME: 'test',
  GIT_COMMITTER_EMAIL: 'test@test'
};

function getBinaries() {
  const beans = process.env.BEANS_BINARY;
  const beansServe = process.env.BEANS_SERVE_BINARY;
  if (!beans || !beansServe) {
    throw new Error('BEANS_BINARY and BEANS_SERVE_BINARY must be set — run tests via e2e/run.sh');
  }
  return { beans, beansServe };
}

/**
 * Create a beans template directory via `beans init`.
 * Called once per worker. Each test gets a fresh git repo but copies the
 * pre-initialized .beans directory to avoid the expensive CLI invocation.
 */
function createBeansTemplate(beansBin: string): string {
  const templateDir = mkdtempSync(join(tmpdir(), 'beans-e2e-template-'));
  execFileSync('git', ['init', '-b', 'main'], { cwd: templateDir, timeout: 10_000 });
  execFileSync('git', ['commit', '--allow-empty', '-m', 'init'], {
    cwd: templateDir,
    timeout: 10_000,
    env: GIT_ENV
  });
  execFileSync(beansBin, ['init'], {
    cwd: templateDir,
    encoding: 'utf-8',
    timeout: 10_000
  });
  return templateDir;
}

/**
 * Wait for a server to start accepting TCP connections, then verify HTTP.
 * Uses TCP connect first (faster than full HTTP fetch for early polls).
 */
async function waitForServer(port: number, timeoutMs = 10_000): Promise<void> {
  const start = Date.now();
  // First, wait for TCP port to open (much faster than HTTP fetch)
  while (Date.now() - start < timeoutMs) {
    const connected = await new Promise<boolean>((resolve) => {
      const socket = connect(port, '127.0.0.1');
      socket.once('connect', () => {
        socket.destroy();
        resolve(true);
      });
      socket.once('error', () => {
        socket.destroy();
        resolve(false);
      });
    });
    if (connected) break;
    await new Promise((r) => setTimeout(r, 25));
  }
  // Then verify HTTP is actually ready
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(`http://localhost:${port}/`);
      if (res.ok) return;
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 50));
  }
  throw new Error(`Server on port ${port} did not start within ${timeoutMs}ms`);
}

/**
 * Helper to run beans CLI commands against a specific beans path.
 */
class BeansCLI {
  constructor(
    readonly beansPath: string,
    readonly projectDir: string,
    private binaryPath: string,
    readonly baseURL: string
  ) {}

  run(args: string[]): string {
    return execFileSync(this.binaryPath, ['--beans-path', this.beansPath, ...args], {
      cwd: this.projectDir,
      encoding: 'utf-8',
      timeout: 10_000
    });
  }

  create(title: string, opts: { type?: string; status?: string; priority?: string } = {}): string {
    const args = ['create', '--json', title, '-t', opts.type ?? 'task'];
    if (opts.status) args.push('-s', opts.status);
    if (opts.priority) args.push('-p', opts.priority);
    const output = this.run(args);
    const json = JSON.parse(output);
    return (json.bean?.id ?? json.id) as string;
  }

  update(id: string, opts: { status?: string; priority?: string; type?: string }): void {
    const args = ['update', id];
    if (opts.status) args.push('-s', opts.status);
    if (opts.priority) args.push('--priority', opts.priority);
    if (opts.type) args.push('-t', opts.type);
    this.run(args);
  }

  /** Run a GraphQL query against the running beans-serve instance. */
  async graphql<T = unknown>(query: string): Promise<T> {
    const res = await fetch(`${this.baseURL}/api/graphql`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query })
    });
    const json = await res.json();
    return json.data as T;
  }

  /** Get all worktree paths from the running server. */
  async getWorktrees(): Promise<{ id: string; path: string; branch: string }[]> {
    const data = await this.graphql<{ worktrees: { id: string; path: string; branch: string }[] }>(
      '{ worktrees { id path branch } }'
    );
    return data.worktrees;
  }

  /** Create a file in a worktree directory to simulate uncommitted changes. */
  createFileInWorktree(worktreePath: string, filename: string, content: string): void {
    writeFileSync(join(worktreePath, filename), content);
  }

  /** Create a commit in a worktree directory to simulate unmerged commits. */
  commitInWorktree(worktreePath: string, filename: string, content: string): void {
    writeFileSync(join(worktreePath, filename), content);
    execFileSync('git', ['add', filename], { cwd: worktreePath, timeout: 10_000 });
    execFileSync('git', ['commit', '-m', 'test commit'], {
      cwd: worktreePath,
      timeout: 10_000,
      env: GIT_ENV
    });
  }
}

type Fixtures = {
  beans: BeansCLI;
  backlogPage: BacklogPage;
  boardPage: BoardPage;
};

type WorkerFixtures = {
  beansTemplate: string;
};

/**
 * Each test gets its own temp directory (copied from a worker-scoped template),
 * beans-serve process, and port. Full isolation — no shared state between tests.
 */
export const test = base.extend<Fixtures, WorkerFixtures>({
  // Worker-scoped: run `beans init` once per worker, copy .beans dir for each test.
  // Fresh git repos are created per test (avoids stale git index/worktree state).
  beansTemplate: [
    async ({}, use) => {
      const { beans: beansBin } = getBinaries();
      const templateDir = createBeansTemplate(beansBin);
      await use(templateDir);
      rmSync(templateDir, { recursive: true, force: true });
    },
    { scope: 'worker' }
  ],

  beans: async ({ page, beansTemplate }, use, testInfo) => {
    const { beans: beansBin, beansServe } = getBinaries();

    // Fresh git repo per test (needed for worktree operations)
    const projectDir = mkdtempSync(join(tmpdir(), 'beans-e2e-'));
    execFileSync('git', ['init', '-b', 'main'], { cwd: projectDir, timeout: 10_000 });
    execFileSync('git', ['commit', '--allow-empty', '-m', 'init'], {
      cwd: projectDir,
      timeout: 10_000,
      env: GIT_ENV
    });
    // Copy pre-initialized beans files from template (avoids expensive `beans init` per test)
    cpSync(join(beansTemplate, '.beans'), join(projectDir, '.beans'), { recursive: true });
    cpSync(join(beansTemplate, '.beans.yml'), join(projectDir, '.beans.yml'));

    const beansPath = join(projectDir, '.beans');

    // Pick a unique port based on worker + test index
    const port = BASE_PORT + testInfo.workerIndex * 100 + testInfo.parallelIndex;

    // Start beans-serve
    const server: ChildProcess = spawn(
      beansServe,
      ['--port', String(port), '--beans-path', beansPath],
      {
        cwd: projectDir,
        env: { ...process.env, GIN_MODE: 'release' },
        stdio: 'pipe'
      }
    );

    try {
      await waitForServer(port);

      // Set the base URL for this test's page
      await page.goto(`http://localhost:${port}/`);
      // Navigate away so tests start fresh with goto()
      await page.goto('about:blank');

      const cli = new BeansCLI(beansPath, projectDir, beansBin, `http://localhost:${port}`);
      await use(cli);
    } finally {
      server.kill();
      rmSync(projectDir, { recursive: true, force: true });
    }
  },

  backlogPage: async ({ page, beans }, use) => {
    const backlog = new BacklogPage(page, beans.baseURL);
    await use(backlog);
  },

  boardPage: async ({ page, beans }, use) => {
    const board = new BoardPage(page, beans.baseURL);
    await use(board);
  }
});

export { expect } from '@playwright/test';
