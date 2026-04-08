import { test, expect } from '@playwright/test';

test.describe('Login page', () => {
  test('shows login form', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/username/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /log in|sign in|login/i })).toBeVisible();
  });

  test('shows error for invalid credentials', async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel(/username/i).fill('nonexistentuser');
    await page.getByLabel(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /log in|sign in|login/i }).click();
    await expect(page.locator('body')).toContainText(/invalid|incorrect|wrong|failed/i);
  });

  test('navigates to register page from login page', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('link', { name: /register|sign up|create/i }).first().click();
    await expect(page).toHaveURL(/register/);
  });
});
