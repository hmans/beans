import { test as base } from '@playwright/test';
import { type ChildProcess, execFileSync, spawn } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
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
		private binaryPath: string,
		readonly baseURL: string
	) {}

	run(args: string[]): string {
		return execFileSync(this.binaryPath, ['--beans-path', this.beansPath, ...args], {
			cwd: PROJECT_ROOT,
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

		// Create isolated temp directory
		const beansPath = mkdtempSync(join(tmpdir(), 'beans-e2e-'));

		// Initialize beans directory
		execFileSync(beansBin, ['--beans-path', beansPath, 'init'], {
			cwd: PROJECT_ROOT,
			encoding: 'utf-8',
			timeout: 10_000
		});

		// Pick a unique port based on worker + test index
		const port = BASE_PORT + testInfo.workerIndex * 100 + testInfo.parallelIndex;

		// Start beans-serve
		const server: ChildProcess = spawn(
			beansServe,
			['--port', String(port), '--beans-path', beansPath],
			{
				cwd: PROJECT_ROOT,
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

			const cli = new BeansCLI(beansPath, beansBin, `http://localhost:${port}`);
			await use(cli);
		} finally {
			server.kill();
			rmSync(beansPath, { recursive: true, force: true });
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
