module.exports = {
  preset: 'react-native',
  testMatch: ['**/__tests__/**/*.test.ts?(x)'],
  modulePathIgnorePatterns: ['<rootDir>/e2e/'],
  transformIgnorePatterns: ['node_modules/(?!(react-native|@react-native)/)'],
  // قرارداد تولیدشده بیرون از پوشهٔ اپ است؛ مسیر `@contracts` همان است که metro و
  // tsconfig تعریف کرده‌اند تا سه محیط (tsc/jest/metro) یک مرجع داشته باشند.
  moduleNameMapper: {
    '^@contracts/(.*)$': '<rootDir>/../../contracts/generated/$1',
  },
};
