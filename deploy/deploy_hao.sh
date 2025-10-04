#!/bin/bash

# 离乱输入法部署脚本
# 用途：生成离乱输入法的 RIME 方案并打包发布
# 作者：荒
# 最后更新：$(date +%Y-%m-%d)

set -e  # 遇到错误立即退出
set -u  # 使用未定义的变量时报错

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >&2
}

error() {
    log "错误: $1" >&2
    exit 1
}

# 检测操作系统类型
OS_TYPE=$(uname)

# 初始化环境变量和工作目录
#source ~/.zshrc

cd "$(dirname $0)" || error "无法切换到脚本目录"
WD="$(pwd)"
SCHEMAS="../schemas"
REF_NAME="${REF_NAME:-v$(date +%Y%m%d%H%M)}"

# 创建本地临时目录
create_ramdisk() {
    RAMDISK="./tmp"
    
    # 清理并重新创建临时目录
    rm -rf "${RAMDISK}"
    mkdir -p "${RAMDISK}" || error "无法创建临时目录"
    
    log "成功创建本地临时目录: ${RAMDISK}"
}

# 清理和准备目录
rm -rf "${SCHEMAS}/ll/build" "${SCHEMAS}/releases"
create_ramdisk
mkdir -p "${SCHEMAS}/releases"

# 生成输入方案
gen_schema() {
    local NAME="$1"
    local DESC="${2:-${NAME}}"
    
    if [ -z "${NAME}" ]; then
        error "方案名称不能为空"
    fi
    
    log "开始生成方案: ${NAME}"
    
    local LL="${RAMDISK}"
    # 设置环境变量
    export SCHEMAS_DIR="${LL}"
    export ASSETS_DIR="${LL}"
    mkdir -p "${LL}" || error "无法创建必要目录"

    # 复制基础文件到内存
    log "复制基础文件到内存..."
    #cp ../table/*.txt "${LL}" || error "复制码表文件失败"
    cp ../template/*.yaml "${LL}" || error "复制模板文件失败"
    cp -r ../template/lua "${LL}/lua" || error "复制 Lua 脚本失败"
    cp -r ../template/opencc "${LL}/opencc" || error "复制 OpenCC 配置失败"
    # 使用自定义配置覆盖默认值
    if [ -d "${NAME}" ]; then
        log "应用自定义配置..."
        cp -r "${NAME}"/*.txt "${LL}"
    fi

    log "生成离乱码表..."
    ./gen_ll -q \
        -d "${LL}/ll_div.txt" \
        -m "${LL}/ll_map.txt" \
        -w "${LL}/ll_words.txt" \
        -f "${LL}/freq.txt" \
        -l "1:4,2:4,3:0,4:0" \
        -L "1:4,2:4,3:4,4:0" \
        -u "${LL}/code_chars_full.txt" \
        -s "${LL}/code_chars_simp.txt" \
        -W "${LL}/code_words_full.txt" \
        -S "${LL}/code_words_simp.txt" \
        -o "${LL}/div_ll.txt" \
        -Z "${LL}/大竹_chai.txt" \
        -C \
        -c "${LL}/ll_citi_pre.txt" \
        -g "${LL}/跟打词提.txt" \
        -z "${LL}/大竹_code.txt" \
        -P "${LL}/lua/chars_cand/preset_data.txt" \
        || error "生成离乱码表失败"

    log "准备生成Rime方案..."
    rsync -a --exclude='/code_*.txt' \
        --exclude='/div_ll.txt' \
        --exclude='/freq.txt' \
        --exclude='/ll_citi_pre.txt' \
        --exclude='/ll_div.txt' \
        --exclude='/ll_map.txt' \
        --exclude='/ll_words.txt' \
        "${LL}/" "${SCHEMAS}/${NAME}/" || error "复制文件失败"
}

# 主程序
log "开始部署离乱输入法..."
gen_schema ll || error "生成离乱方案失败"
log "部署完成"
