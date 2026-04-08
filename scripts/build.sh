#!/bin/bash
# Mycel Mesh 跨平台编译打包脚本
# 使用方法：./scripts/build.sh

set -e

VERSION="${VERSION:-1.0.0}"
OUTPUT_DIR="./dist"

echo "========================================"
echo "Mycel Mesh v${VERSION} 编译打包"
echo "========================================"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 定义目标平台
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# 编译函数
build_platform() {
    local os=$1
    local arch=$2
    local ext=""
    local os_name="${os}"

    if [ "$os" = "windows" ]; then
        ext=".exe"
        os_name="win"
    fi

    local artifact_name="mycel-${os_name}-${arch}"
    local output_path="${OUTPUT_DIR}/${artifact_name}"

    echo ""
    echo ">>> 编译 ${os}/${arch} ..."

    mkdir -p "$output_path"

    # 设置环境变量
    export GOOS="$os"
    export GOARCH="$arch"

    # 编译 Coordinator
    echo "  - 编译 coordinator${ext}"
    go build -ldflags "-X main.Version=${VERSION}" \
        -o "${output_path}/coordinator${ext}" \
        ./cmd/coordinator

    # 编译 Agent
    echo "  - 编译 agent${ext}"
    go build -ldflags "-X main.Version=${VERSION}" \
        -o "${output_path}/agent${ext}" \
        ./cmd/agent

    # 编译 CLI
    echo "  - 编译 mycelctl${ext}"
    go build -ldflags "-X main.Version=${VERSION}" \
        -o "${output_path}/mycelctl${ext}" \
        ./cmd/mycelctl

    # 复制配置文件示例
    cp docs/quickstart.md "$output_path/QUICKSTART.md" 2>/dev/null || true
    cp README.md "$output_path/README.md" 2>/dev/null || true

    # 重置环境变量
    unset GOOS
    unset GOARCH

    echo "  ✓ ${artifact_name} 完成"
}

# 编译所有平台
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "$platform"
    build_platform "$os" "$arch"
done

# 创建源码压缩包
echo ""
echo ">>> 创建源码包 ..."
tar -czf "${OUTPUT_DIR}/mycel-${VERSION}-source.tar.gz" \
    --exclude=".git" \
    --exclude="dist" \
    --exclude="bin" \
    --exclude=".idea" \
    -C . .

echo ""
echo "========================================"
echo "编译完成！"
echo "========================================"
echo ""
echo "输出目录：${OUTPUT_DIR}/"
ls -la "$OUTPUT_DIR"
echo ""
echo "版本：v${VERSION}"
