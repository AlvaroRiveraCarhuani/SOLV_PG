// ESLint flat config for the SOLV_PG frontend.
// package.json has no "type" field (CommonJS), so the flat config uses the .mjs suffix.
// Scope: TypeScript (typescript-eslint recommended, non-type-checked) and Angular
// templates (angular-eslint recommended). Correctness, security and template-safety
// rules are never disabled here; only generated output is ignored.
import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import angular from 'angular-eslint';

export default tseslint.config(
  {
    ignores: [
      'dist/**',
      'coverage/**',
      '.angular/**',
      'test-results/**',
      'playwright-report/**',
      'node_modules/**',
    ],
  },
  {
    files: ['**/*.ts'],
    extends: [
      eslint.configs.recommended,
      ...tseslint.configs.recommended,
      ...angular.configs.tsRecommended,
    ],
    processor: angular.processInlineTemplates,
    rules: {
      // Params intentionally unused by the body but bound positionally from
      // templates keep the leading-underscore convention (Angular template
      // bindings are positional, so renaming is safe and lint-clean).
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
    },
  },
  {
    files: ['**/*.html'],
    extends: [...angular.configs.templateRecommended],
    rules: {},
  },
);
