#!/usr/bin/env node

const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');

console.log('📦 正在准备 npm 包...');

// 创建 binaries 目录
const binariesDir = path.join(__dirname, '..', 'binaries');
if (!fs.existsSync(binariesDir)) {
  fs.mkdirSync(binariesDir, { recursive: true });
}

console.log('🔨 正在编译 Go 二进制文件...');

const platforms = [
  { goos: 'darwin', goarch: 'amd64', output: 'skillctl-cli-darwin-amd64' },
  { goos: 'darwin', goarch: 'arm64', output: 'skillctl-cli-darwin-arm64' },
  { goos: 'linux', goarch: 'amd64', output: 'skillctl-cli-linux-amd64' },
  { goos: 'linux', goarch: 'arm64', output: 'skillctl-cli-linux-arm64' },
  { goos: 'windows', goarch: 'amd64', output: 'skillctl-cli-windows-amd64.exe' },
];

const rootDir = path.join(__dirname, '..');

platforms.forEach(({ goos, goarch, output }) => {
  try {
    console.log(`  🔨 编译 ${goos}/${goarch}...`);
    const outputPath = path.join(binariesDir, output);
    
    execSync(
      `go build -ldflags="-s -w" -o "${outputPath}" .`,
      {
        cwd: rootDir,
        env: { ...process.env, GOOS: goos, GOARCH: goarch },
        stdio: 'pipe'
      }
    );
    
    // 设置可执行权限（非 Windows）
    if (!output.endsWith('.exe')) {
      fs.chmodSync(outputPath, '755');
    }
    
    console.log(`  ✅ ${output}`);
  } catch (err) {
    console.error(`  ❌ 编译失败 ${goos}/${goarch}: ${err.message}`);
  }
});

console.log('✅ npm 包准备完成！');
