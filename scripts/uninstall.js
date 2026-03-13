#!/usr/bin/env node

const path = require('path');
const fs = require('fs');

console.log('🗑️  正在卸载 skillctl...');

const binariesDir = path.join(__dirname, '..', 'binaries');

// 删除 binaries 目录
if (fs.existsSync(binariesDir)) {
  try {
    fs.rmSync(binariesDir, { recursive: true, force: true });
    console.log('✅ skillctl 已卸载');
  } catch (err) {
    console.warn(`⚠️  无法删除 binaries 目录: ${err.message}`);
  }
} else {
  console.log('✅ skillctl 已经卸载');
}