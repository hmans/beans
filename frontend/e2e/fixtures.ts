import { test as base } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { readdirSync, unlinkSync } from 'node:fs';
import { join } from 'node:path';
import { BacklogPage } from './pages/backlog-page';
import { BoardPage } from './pages/board-page';

const PROJECT_ROOT = join(import.meta.dirname, '../..');

/**
 * Helper to run beans CLI commands against a specific beans path.
 * Uses execFileSync (no shell) to avoid command injection.
 */
class BeansCLI {
	constructor(readonly beansPath: string) {}

	run(args: string[]): string {
		return execFileSync(
			'mise',
			['exec', '--', 'go', 'run', './cmd/beans', '--beans-path', this.beansPath, ...args],
			{
				cwd: PROJECT_ROOT,
				encoding: 'utf-8',
				timeout: 15_000
			}
		);
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

	/** Delete all bean .md files from the beans path. */
	deleteAll(): void {
		const files = readdirSync(this.beansPath);
		for (const file of files) {
			if (file.endsWith('.md')) {
				unlinkSync(join(this.beansPath, file));
			}
		}
	}
}

type Fixtures = {
	beans: BeansCLI;
	backlogPage: BacklogPage;
	boardPage: BoardPage;
};

/**
 * Custom test fixture providing a BeansCLI and page objects.
 *
 * The beans-serve webServer reads BEANS_E2E_PATH from the environment.
 * A wrapper script creates the temp dir and sets the env var before
 * starting Playwright.
 *
 * Each test gets a clean slate: all bean files are deleted before the test runs.
 */
export const test = base.extend<Fixtures>({
	beans: async ({}, use) => {
		const beansPath = process.env.BEANS_E2E_PATH;
		if (!beansPath) {
			throw new Error('BEANS_E2E_PATH not set — run tests via the e2e script');
		}
		const cli = new BeansCLI(beansPath);
		// Clean slate for each test
		cli.deleteAll();
		await use(cli);
	},

	backlogPage: async ({ page }, use) => {
		await use(new BacklogPage(page));
	},

	boardPage: async ({ page }, use) => {
		await use(new BoardPage(page));
	}
});

export { expect } from '@playwright/test';
