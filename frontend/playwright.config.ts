import { defineConfig } from '@playwright/test';

const PORT = 22881;

export default defineConfig({
	testDir: './e2e',
	timeout: 30_000,
	retries: 0,
	/* Tests share a single server and beans dir, so run serially */
	workers: 1,
	use: {
		baseURL: `http://localhost:${PORT}`,
		trace: 'on-first-retry'
	},
	webServer: {
		command: `mise exec -- go run ../cmd/beans-serve --port ${PORT} --beans-path $BEANS_E2E_PATH`,
		port: PORT,
		reuseExistingServer: false,
		timeout: 30_000,
		env: {
			GIN_MODE: 'release'
		}
	}
});
