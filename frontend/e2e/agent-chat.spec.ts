import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect } from './fixtures';

test.describe('Agent chat', () => {
	test('Clear button resets the conversation in the UI', async ({ page, beans }) => {
		// Seed a conversation file directly — the agent manager loads from disk
		// when a subscription connects with no in-memory session, so no Claude
		// process is needed.
		const convDir = join(beans.beansPath, '.conversations');
		mkdirSync(convDir, { recursive: true });
		writeFileSync(
			join(convDir, '__central__.jsonl'),
			[
				JSON.stringify({ type: 'message', role: 'user', content: 'hello agent' }),
				JSON.stringify({ type: 'message', role: 'assistant', content: 'Hi! How can I help?' })
			].join('\n') + '\n'
		);

		await page.goto(beans.baseURL + '/');

		// Open the agent chat panel
		await page.click('button[title="Show chat"]');

		// Verify the seeded messages are visible
		await expect(page.locator('text=hello agent')).toBeVisible({ timeout: 5000 });
		await expect(page.locator('text=Hi! How can I help?')).toBeVisible({ timeout: 5000 });

		// Clear button should be enabled
		const clearBtn = page.locator('button:has-text("Clear")');
		await expect(clearBtn).toBeEnabled();

		// Click Clear
		await clearBtn.click();

		// The empty state message should reappear
		await expect(page.locator('text=Send a message to start a conversation')).toBeVisible({
			timeout: 5000
		});

		// The messages should be gone
		await expect(page.locator('text=hello agent')).not.toBeVisible();
		await expect(page.locator('text=Hi! How can I help?')).not.toBeVisible();

		// Clear button should be disabled again
		await expect(clearBtn).toBeDisabled();
	});
});
