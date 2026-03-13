#!/usr/bin/env node

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// 配置
const REPO_OWNER = 'your-username';
const REPO_NAME = 'skillctl';
const BINARY_NAME = 'skillctl';

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

const targetPlatform = platformMap[platform];
const targetArch = archMap[arch];

if (!targetPlatform || !targetArch) {
  console.error(`❌ 不支持的平台: ${platform}/${arch}`);
  process.exit(1);
}

console.log(`🚀 正在安装 skillctl...`);
console.log(`📦 目标平台: ${targetPlatform}/${targetArch}`);

// 构建下载 URL
const binaryName = platform === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME;
const downloadURL = `https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${BINARY_NAME}-${targetPlatform}-${targetArch}${platform === 'win32' ? '.exe' : ''}`;

console.log(`⬇️  下载地址: ${downloadURL}`);

// 创建 binaries 目录
const binariesDir = path.join(__dirname, '..', 'binaries');
if (!fs.existsSync(binariesDir)) {
  fs.mkdirSync(binariesDir, { recursive: true });
}

const outputPath = path.join(binariesDir, binaryName);

// 下载文件
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const protocol = url.startsWith('https') ? https : http;

    protocol.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // 处理重定向
        downloadFile(response.headers.location, dest)
          .then(resolve)
          .catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`下载失败: HTTP ${response.statusCode}`));
        return;
      }

      const totalSize = parseInt(response.headers['content-length'], 10);
      let downloadedSize = 0;

      response.pipe(file);

      response.on('data', (chunk) => {
        downloadedSize += chunk.length;
        if (totalSize) {
          const progress = Math.round((downloadedSize / totalSize) * 100);
          process.stdout.write(`\r📥 下载进度: ${progress}%`);
        }
      });

      file.on('finish', () => {
        file.close();
        console.log('\n✅ 下载完成');
        resolve();
      });

      file.on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    }).on('error', (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

// 主安装流程
async function install() {
  try {
    // 检查是否已经安装
    if (fs.existsSync(outputPath)) {
      console.log('✅ skillctl 已经安装');
      return;
    }

    // 下载二进制文件
    await downloadFile(downloadURL, outputPath);

    // 设置可执行权限（非 Windows）
    if (platform !== 'win32') {
      try {
        fs.chmodSync(outputPath, '755');
      } catch (err) {
        console.warn('⚠️  无法设置可执行权限');
      }
    }

    // 验证安装
    try {
      execSync(`"${outputPath}" --version`, { stdio: 'pipe' });
      console.log('✅ skillctl 安装成功！');
    } catch (err) {
      console.error('❌ 安装验证失败');
      fs.unlinkSync(outputPath);
      process.exit(1);
    }

  } catch (error) {
    console.error(`❌ 安装失败: ${error.message}`);
    if (fs.existsSync(outputPath)) {
      fs.unlinkSync(outputPath);
    }
    process.exit(1);
  }
}

install();