import { defineConfig } from '@playwright/test';

export default defineConfig({
	testDir: './e2e',
	timeout: 30_000,
	retries: 0,
	workers: 4,
	use: {
		trace: 'on-first-retry'
	}
	// No webServer — each test spawns its own beans-serve via fixtures
});
