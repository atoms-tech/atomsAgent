module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  testMatch: ['**/__tests__/**/*.test.ts'],
  modulePathIgnorePatterns: ['dist'],
  setupFilesAfterEnv: ['<rootDir>/jest.setup.cjs'],
};
