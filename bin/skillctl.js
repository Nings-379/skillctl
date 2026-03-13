#!/usr/bin/env node

const path = require('path');
const { spawnSync } = require('child_process');
const os = require('os');

// 获取平台和架构
const platform = os.platform();
const arch = os.arch();

// 映射平台名称
const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows'
};

// 映射架构名称
const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64',
  'ia32': '386'
};

// 构建可执行文件名
const binaryName = platform === 'win32' ? 'skillctl-cli.exe' : 'skillctl-cli';
const binaryPath = path.join(__dirname, 'binaries', binaryName);

// 检查二进制文件是否存在
if (!require('fs').existsSync(binaryPath)) {
  console.error('❌ 错误: skillctl-cli 二进制文件不存在');
  console.error('📝 请运行: npm install skillctl-cli');
  process.exit(1);
}

// 执行二进制文件
const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: { ...process.env }
});

// 退出码
process.exit(result.status || 0);