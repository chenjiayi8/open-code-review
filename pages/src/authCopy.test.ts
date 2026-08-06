import { describe, expect, test } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const repoRoot = join(__dirname, '..');

function collectFiles(dir: string, extensions: Set<string>): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const fullPath = join(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) return collectFiles(fullPath, extensions);
    if (!extensions.has(fullPath.slice(fullPath.lastIndexOf('.')))) return [];
    return [fullPath];
  });
}

describe('native runner auth copy', () => {
  const copyFiles = [
    ...collectFiles(join(repoRoot, 'src', 'content', 'docs'), new Set(['.md', '.mdx'])),
    ...collectFiles(join(repoRoot, 'src', 'i18n'), new Set(['.ts'])),
    join(repoRoot, 'src', 'components', 'QuickStartSection.tsx'),
  ];

  test('does not present OCR credentials, subscriptions, or claude.ai-only login as setup requirements', () => {
    const banned = [
      /subscription/i,
      /claude\.ai/i,
      /--claudeai/i,
      /OCR API key/i,
      /OCR API 密钥/i,
      /OCR 側の資格情報設定/i,
      /учетных данных OCR/i,
    ];

    const matches = copyFiles.flatMap((file) => {
      const content = readFileSync(file, 'utf8');
      return banned
        .filter((pattern) => pattern.test(content))
        .map((pattern) => `${relative(repoRoot, file)} matched ${pattern}`);
    });

    expect(matches).toEqual([]);
  });
});
