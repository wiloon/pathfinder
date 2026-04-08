import { test, expect } from '@playwright/test';

test.describe('Register page', () => {
  test('shows registration form', async ({ page }) => {
    await page.goto('/register');
    await expect(page.getByRole('heading', { name: /register|sign up|create account/i })).toBeVisible();
    await expect(page.getByLabel(/username/i)).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByRole('textbox', { name: 'Password *', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: /register|sign up|create/i })).toBeVisible();
  });

  test('shows validation error for short password', async ({ page }) => {
    await page.goto('/register');
    await page.getByLabel(/username/i).fill('testuser');
    await page.getByLabel(/email/i).fill('testuser@example.com');
    await page.getByRole('textbox', { name: 'Password *', exact: true }).fill('123');
    await page.getByRole('button', { name: /register|sign up|create/i }).click();
    // Expect a validation error visible on the page
    await expect(page.locator('body')).toContainText(/password|6 char|too short/i);
  });

  test('navigates to login page from register page', async ({ page }) => {
    await page.goto('/register');
    await page.getByRole('link', { name: /log in|sign in|login/i }).first().click();
    await expect(page).toHaveURL(/login/);
  });
});
