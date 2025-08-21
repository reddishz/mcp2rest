#!/bin/bash

# MCP2REST 进程监控脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查进程数量
check_process_count() {
    local count=$(pgrep -f "mcp2rest" | wc -l)
    echo $count
}

# 显示进程详情
show_processes() {
    echo "当前运行的 mcp2rest 进程:"
    echo "================================"
    if pgrep -f "mcp2rest" > /dev/null; then
        ps aux | grep mcp2rest | grep -v grep | while read line; do
            echo "$line"
        done
    else
        echo "没有发现 mcp2rest 进程"
    fi
    echo "================================"
}

# 显示进程树
show_process_tree() {
    echo "进程树:"
    echo "================================"
    if pgrep -f "mcp2rest" > /dev/null; then
        pstree -p $(pgrep -f "mcp2rest" | head -1) 2>/dev/null || echo "无法显示进程树"
    else
        echo "没有发现 mcp2rest 进程"
    fi
    echo "================================"
}

# 清理所有进程
cleanup_processes() {
    local count=$(check_process_count)
    if [ $count -gt 0 ]; then
        log_warn "发现 $count 个 mcp2rest 进程，正在清理..."
        pkill -f "mcp2rest"
        sleep 2
        
        # 强制清理
        local remaining=$(check_process_count)
        if [ $remaining -gt 0 ]; then
            log_warn "仍有 $remaining 个进程，强制清理..."
            pkill -9 -f "mcp2rest"
            sleep 1
        fi
        
        local final_count=$(check_process_count)
        if [ $final_count -eq 0 ]; then
            log_info "所有进程已清理"
        else
            log_error "仍有 $final_count 个进程无法清理"
        fi
    else
        log_info "没有发现需要清理的进程"
    fi
}

# 监控模式
monitor_mode() {
    log_info "进入监控模式，按 Ctrl+C 退出"
    while true; do
        local count=$(check_process_count)
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        echo "[$timestamp] 进程数量: $count"
        
        if [ $count -gt 5 ]; then
            log_warn "进程数量过多: $count"
        fi
        
        sleep 5
    done
}

# 主函数
main() {
    case "${1:-status}" in
        "status")
            local count=$(check_process_count)
            log_info "当前 mcp2rest 进程数量: $count"
            show_processes
            ;;
        "tree")
            show_process_tree
            ;;
        "cleanup")
            cleanup_processes
            ;;
        "monitor")
            monitor_mode
            ;;
        "help"|"-h"|"--help")
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  status    - 显示当前进程状态 (默认)"
            echo "  tree      - 显示进程树"
            echo "  cleanup   - 清理所有 mcp2rest 进程"
            echo "  monitor   - 持续监控进程数量"
            echo "  help      - 显示此帮助信息"
            ;;
        *)
            log_error "未知命令: $1"
            echo "使用 '$0 help' 查看帮助"
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
