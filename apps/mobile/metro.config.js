/**
 * تنظیمات Metro.
 *
 * پوشهٔ `contracts/` بیرون از اپ است (منبع مشترک قرارداد سرور و کلاینت)، پس باید در
 * watchFolders و resolver بیاید؛ در غیر این صورت import انواع تولیدشده در باندل
 * شکست می‌خورد.
 */
const path = require('path');
const { getDefaultConfig, mergeConfig } = require('@react-native/metro-config');

const workspaceRoot = path.resolve(__dirname, '../..');

/** @type {import('metro-config').MetroConfig} */
const config = {
  watchFolders: [path.resolve(workspaceRoot, 'contracts')],
  resolver: {
    nodeModulesPaths: [path.resolve(__dirname, 'node_modules')],
    extraNodeModules: {
      '@contracts': path.resolve(workspaceRoot, 'contracts/generated'),
    },
  },
};

module.exports = mergeConfig(getDefaultConfig(__dirname), config);
