#!/usr/bin/env node

const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');

console.log('🔧 正在配置 skillctl-cli...');

const binariesDir = path.join(__dirname, '..', 'binaries');
const binaryName = process.platform === 'win32' ? 'skillctl.exe' : 'skillctl';
const binaryPath = path.join(binariesDir, binaryName);

// 检查二进制文件是否存在
if (fs.existsSync(binaryPath)) {
  try {
    // 验证版本
    const version = execSync(`"${binaryPath}" --version`, { encoding: 'utf8' }).trim();
    console.log(`✅ skillctl-cli ${version} 已就绪`);
  } catch (err) {
    console.warn('⚠️  无法验证 skillctl-cli 版本');
  }
} else {
  console.warn('⚠️  skillctl 二进制文件不存在，请运行 npm install skillctl-cli');
}

console.log('🎉 安装完成！');