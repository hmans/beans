import { expect, type Locator, type Page } from '@playwright/test';

/**
 * Page object for the board (kanban) view at /board.
 */
export class BoardPage {
	constructor(
		private page: Page,
		private baseURL: string
	) {}

	async goto() {
		await this.page.goto(this.baseURL + '/board');
		// Wait for columns to render
		await this.page.waitForSelector('[data-status]', { timeout: 10_000 });
	}

	/** Get a column locator by status name. */
	private column(status: string): Locator {
		return this.page.locator(`[data-status="${status}"]`);
	}

	/** Get all bean titles in a specific column, in display order. */
	async getColumnTitles(status: string): Promise<string[]> {
		const col = this.column(status);
		const cards = col.locator('[role="listitem"] button span.text-sm');
		const titles = await cards.allTextContents();
		return titles.map((t) => t.trim());
	}

	/** Get the count of beans in a column. */
	async getColumnCount(status: string): Promise<number> {
		const col = this.column(status);
		return col.locator('[role="listitem"]').count();
	}

	/** Wait for a specific count of beans in a column. */
	async waitForColumnCount(status: string, count: number) {
		const col = this.column(status);
		await expect(col.locator('[role="listitem"]')).toHaveCount(count, { timeout: 10_000 });
	}

	/** Wait for a bean to appear in a specific column. */
	async waitForBeanInColumn(title: string, status: string) {
		const col = this.column(status);
		await col.locator('[role="listitem"]', { hasText: title }).waitFor({ timeout: 10_000 });
	}

	/** Wait for a bean to disappear from a specific column. */
	async waitForBeanNotInColumn(title: string, status: string) {
		const col = this.column(status);
		await col
			.locator('[role="listitem"]', { hasText: title })
			.waitFor({ state: 'detached', timeout: 10_000 });
	}
}
