import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect } from './fixtures';
import { agentSession } from './agent-session';

test.describe('Agent chat', () => {
  test('Clear button resets the conversation in the UI', async ({ page, beans }) => {
    await agentSession('__central__', beans)
      .withMessages([
        { role: 'user', content: 'hello agent' },
        { role: 'assistant', content: 'Hi! How can I help?' }
      ])
      .open(page);

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

  test('Image attachments are displayed inline in user messages', async ({ page, beans }) => {
    // Create a tiny 1x1 red PNG for testing
    const pngData = Buffer.from(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==',
      'base64'
    );
    const imageId = 'test-image.png';

    // Write the image file to the attachments directory
    const attachDir = join(beans.beansPath, '.conversations', 'attachments', '__central__');
    mkdirSync(attachDir, { recursive: true });
    writeFileSync(join(attachDir, imageId), pngData);

    // Seed a conversation with an image reference
    await agentSession('__central__', beans)
      .withMessages([
        {
          role: 'user',
          content: 'Check this screenshot',
          images: [{ id: imageId, media_type: 'image/png' }]
        },
        { role: 'assistant', content: 'I can see the image.' }
      ])
      .open(page);

    // Verify the text message is visible
    await expect(page.locator('text=Check this screenshot')).toBeVisible({ timeout: 5000 });

    // Verify the image is rendered inline
    const img = page.locator('img[alt="Attached image"]');
    await expect(img).toBeVisible({ timeout: 5000 });

    // Verify the image src points to the attachment endpoint
    const src = await img.getAttribute('src');
    expect(src).toContain('/api/attachments/__central__/test-image.png');
  });
});
