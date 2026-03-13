import { test as base } from '@playwright/test';
import { type ChildProcess, execFileSync, spawn } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { BacklogPage } from './pages/backlog-page';
import { BoardPage } from './pages/board-page';

const PROJECT_ROOT = join(import.meta.dirname, '../..');
const BASE_PORT = 22900;

function getBinaries() {
  const beans = process.env.BEANS_BINARY;
  const beansServe = process.env.BEANS_SERVE_BINARY;
  if (!beans || !beansServe) {
    throw new Error('BEANS_BINARY and BEANS_SERVE_BINARY must be set — run tests via e2e/run.sh');
  }
  return { beans, beansServe };
}

/**
 * Wait for a server to start accepting connections.
 */
async function waitForServer(port: number, timeoutMs = 10_000): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(`http://localhost:${port}/`);
      if (res.ok) return;
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 100));
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
      env: {
        ...process.env,
        GIT_AUTHOR_NAME: 'test',
        GIT_AUTHOR_EMAIL: 'test@test',
        GIT_COMMITTER_NAME: 'test',
        GIT_COMMITTER_EMAIL: 'test@test'
      }
    });
  }
}

type Fixtures = {
  beans: BeansCLI;
  backlogPage: BacklogPage;
  boardPage: BoardPage;
};

/**
 * Each test gets its own temp directory, beans-serve process, and port.
 * Full isolation — no shared state between tests.
 */
export const test = base.extend<Fixtures>({
  beans: async ({ page }, use, testInfo) => {
    const { beans: beansBin, beansServe } = getBinaries();

    // Create isolated temp project directory with a git repo
    const projectDir = mkdtempSync(join(tmpdir(), 'beans-e2e-'));
    execFileSync('git', ['init'], { cwd: projectDir, timeout: 10_000 });
    execFileSync('git', ['commit', '--allow-empty', '-m', 'init'], {
      cwd: projectDir,
      timeout: 10_000,
      env: { ...process.env, GIT_AUTHOR_NAME: 'test', GIT_AUTHOR_EMAIL: 'test@test', GIT_COMMITTER_NAME: 'test', GIT_COMMITTER_EMAIL: 'test@test' }
    });

    // Initialize beans directory inside the project
    execFileSync(beansBin, ['init'], {
      cwd: projectDir,
      encoding: 'utf-8',
      timeout: 10_000
    });

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
