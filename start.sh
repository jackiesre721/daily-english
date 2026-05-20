#!/bin/bash
# Daily English - 启动脚本

set -e

cd "$(dirname "$0")"

APP="daily-english"
BIN="./server"
PORT="${PORT:-8080}"
PID_FILE="./.server.pid"

red()   { printf "\033[31m%s\033[0m\n" "$1"; }
green() { printf "\033[32m%s\033[0m\n" "$1"; }
yellow(){ printf "\033[33m%s\033[0m\n" "$1"; }

build() {
    yellow "Building..."
    go build -o "$BIN" ./cmd/server/
    green "Build OK"
}

stop() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            rm -f "$PID_FILE"
            green "Stopped (PID: $PID)"
        else
            rm -f "$PID_FILE"
            yellow "Stale PID file removed"
        fi
    else
        # Try port-based kill
        PID=$(lsof -ti:"$PORT" 2>/dev/null || true)
        if [ -n "$PID" ]; then
            kill "$PID" 2>/dev/null || true
            green "Stopped process on port $PORT"
        else
            yellow "No running server found"
        fi
    fi
}

start() {
    build
    stop

    echo "Starting on port $PORT..."
    "$BIN" -port "$PORT" &
    echo $! > "$PID_FILE"

    # Wait for server to be ready
    for i in $(seq 1 10); do
        if curl -s "http://localhost:$PORT/api/ping" > /dev/null 2>&1; then
            green "Server running at http://localhost:$PORT"
            green "  Home:   http://localhost:$PORT/home.html"
            green "  Vocab:  http://localhost:$PORT/home.html?tab=vocab"
            echo ""
            echo "  PID: $(cat "$PID_FILE") | Stop: ./start.sh stop"
            return 0
        fi
        sleep 0.5
    done
    red "Server failed to start"
    return 1
}

case "${1:-start}" in
    start)   start ;;
    stop)    stop ;;
    restart) start ;;
    build)   build ;;
    status)
        if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
            green "Running (PID: $(cat "$PID_FILE"), port: $PORT)"
        else
            yellow "Stopped"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|build|status}"
        echo ""
        echo "  start    Build and start server (default)"
        echo "  stop     Stop running server"
        echo "  restart  Stop and start"
        echo "  build    Build only"
        echo "  status   Check if server is running"
        echo ""
        echo "Environment:"
        echo "  PORT     Server port (default: 8080)"
        exit 1
        ;;
esac
