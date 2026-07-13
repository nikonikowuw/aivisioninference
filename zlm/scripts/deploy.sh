#!/bin/bash

# ZLMediaKit Local Deployment Script
# Supports: macOS (Homebrew), Ubuntu/Debian (apt), CentOS/RHEL/Fedora (yum/dnf)

set -e

ZLM_DIR="${ZLM_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
ZLM_BUILD_DIR="$ZLM_DIR/build"
ZLM_RELEASE_DIR="$ZLM_DIR/bin"
ZLM_CONFIG_DIR="$ZLM_DIR/config"
ZLM_REPO="https://github.com/ZLMediaKit/ZLMediaKit.git"

# ── OS detection ──────────────────────────────────────────────
detect_os() {
  case "$(uname -s)" in
    Darwin*)  echo "macos" ;;
    Linux*)   echo "linux" ;;
    *)        echo "unknown" ;;
  esac
}

install_deps_macos() {
  if [ "$EUID" -eq 0 ] && [ -n "$SUDO_USER" ]; then
    echo "Running as root, dropping privileges for Homebrew..."
    sudo -u "$SUDO_USER" brew update 2>/dev/null || echo "brew update skipped (network issue)"
    sudo -u "$SUDO_USER" brew install cmake git openssl sdl2 x264 x265 faac lame opus
    sudo -u "$SUDO_USER" brew install ffmpeg || true
  else
    echo "Installing dependencies via Homebrew..."
    brew update 2>/dev/null || echo "brew update skipped (network issue)"
    brew install cmake git openssl sdl2 x264 x265 faac lame opus
    brew install ffmpeg || true
  fi
}

install_deps_debian() {
  echo "Installing dependencies via apt..."
  sudo apt-get update
  sudo apt-get install -y build-essential cmake git \
    libssl-dev libsdl2-dev libavcodec-dev libavutil-dev \
    libavformat-dev libswscale-dev libx264-dev libx265-dev \
    libfaac-dev libmp3lame-dev libopus-dev
}

install_deps_rhel() {
  echo "Installing dependencies via dnf..."
  sudo dnf install -y cmake git gcc-c++ openssl-devel SDL2-devel \
    ffmpeg-devel x264-devel x265-devel libopus-devel
}

# ── Build ZLM ─────────────────────────────────────────────────
build_zlm() {
  SRC_DIR="$ZLM_DIR/src"

  RUN=""
  if [ "$EUID" -eq 0 ] && [ -n "$SUDO_USER" ] && [ "$(uname -s)" = "Darwin" ]; then
    RUN="sudo -u $SUDO_USER"
  fi

  if [ ! -d "$SRC_DIR" ]; then
    echo "Cloning ZLMediaKit source into $SRC_DIR..."
    $RUN git clone --depth 1 $ZLM_REPO "$SRC_DIR"
  elif [ ! -d "$SRC_DIR/.git" ]; then
    echo "Source directory $SRC_DIR exists but has no .git, re-cloning..."
    rm -rf "$SRC_DIR"
    $RUN git clone --depth 1 $ZLM_REPO "$SRC_DIR"
  else
    echo "Source directory $SRC_DIR already exists, skipping clone."
  fi

  $RUN bash -c "cd '$SRC_DIR' && git submodule update --init --recursive"

  echo "Building ZLMediaKit..."
  $RUN mkdir -p "$ZLM_BUILD_DIR"

  CMAKE_FLAGS="-DCMAKE_INSTALL_PREFIX=$ZLM_RELEASE_DIR"
  if [ "$(uname -s)" = "Darwin" ]; then
    SDK=$(xcrun --show-sdk-path)
    CMAKE_FLAGS="$CMAKE_FLAGS -DCMAKE_OSX_SYSROOT=$SDK"
    CMAKE_FLAGS="$CMAKE_FLAGS -DCMAKE_OSX_ARCHITECTURES=arm64"
  fi
  CMAKE_FLAGS="$CMAKE_FLAGS -DENABLE_WEBRTC=OFF -DENABLE_SRT=OFF"

  $RUN bash -c "cd '$ZLM_BUILD_DIR' && cmake '$SRC_DIR' $CMAKE_FLAGS"
  CORES=$(sysctl -n hw.logicalcpu 2>/dev/null || nproc)
  $RUN bash -c "cd '$ZLM_BUILD_DIR' && make -j$CORES"

  # Copy binary to release directory
  mkdir -p "$ZLM_RELEASE_DIR"
  ZLM_BINARY=$(find "$ZLM_DIR" -path "*/release/*/MediaServer" -type f 2>/dev/null | head -1)
  if [ -n "$ZLM_BINARY" ]; then
    cp "$ZLM_BINARY" "$ZLM_RELEASE_DIR/"
    echo "Binary installed to $ZLM_RELEASE_DIR/MediaServer"
    ls -la "$ZLM_RELEASE_DIR/MediaServer"
  else
    echo "Warning: MediaServer binary not found in build output"
  fi
}

# ── Service setup ─────────────────────────────────────────────
setup_service_macos() {
  if [ "$EUID" -eq 0 ] && [ -n "$SUDO_USER" ]; then
    USER_HOME=$(eval echo ~"$SUDO_USER")
  else
    USER_HOME="$HOME"
  fi
  PLIST_PATH="$USER_HOME/Library/LaunchAgents/local.zlmediakit.plist"
  mkdir -p "$(dirname "$PLIST_PATH")"
  cat <<EOF > "$PLIST_PATH"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>local.zlmediakit</string>
  <key>ProgramArguments</key>
  <array>
    <string>$ZLM_RELEASE_DIR/MediaServer</string>
    <string>-c</string>
    <string>$ZLM_CONFIG_DIR/config.ini</string>
    <string>-d</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$USER_HOME/Library/Logs/zlmediakit.log</string>
  <key>StandardErrorPath</key>
  <string>$USER_HOME/Library/Logs/zlmediakit.log</string>
</dict>
</plist>
EOF
  # Unload existing if present, then load
  launchctl unload "$PLIST_PATH" 2>/dev/null || true
  launchctl load "$PLIST_PATH"
  echo "LaunchAgent installed. Use 'launchctl start local.zlmediakit' to start."
}

setup_service_linux() {
  echo "Setting up Systemd service..."
  sudo bash -c "cat <<EOF > /etc/systemd/system/zlmediakit.service
[Unit]
Description=ZLMediaKit Streaming Server
After=network.target

[Service]
Type=simple
WorkingDirectory=$ZLM_RELEASE_DIR
ExecStart=$ZLM_RELEASE_DIR/MediaServer -c $ZLM_CONFIG_DIR/config.ini
Restart=always
User=root

[Install]
WantedBy=multi-target.target
EOF"
  sudo systemctl daemon-reload
  echo "Systemd service installed. Use 'sudo systemctl start zlmediakit' to start."
}

# ── Main ──────────────────────────────────────────────────────
OS=$(detect_os)
echo "Detected OS: $OS"

case "$OS" in
  macos)
    install_deps_macos
    build_zlm
    setup_service_macos
    if [ "$EUID" -eq 0 ] && [ -n "$SUDO_USER" ]; then
      USER_HOME=$(eval echo ~"$SUDO_USER")
    else
      USER_HOME="$HOME"
    fi
    echo ""
    echo "Deployment complete!"
    echo "Start: launchctl start local.zlmediakit"
    echo "Stop:  launchctl stop local.zlmediakit"
    echo "Logs:  $USER_HOME/Library/Logs/zlmediakit.log"
    ;;
  linux)
    # Distro detection
    if command -v apt-get &>/dev/null; then
      install_deps_debian
    elif command -v dnf &>/dev/null; then
      install_deps_rhel
    elif command -v yum &>/dev/null; then
      install_deps_rhel
    else
      echo "Unsupported Linux distro (neither apt nor dnf/yum found)."
      echo "Please install build dependencies manually, then re-run."
      exit 1
    fi
    build_zlm
    setup_service_linux
    echo ""
    echo "Deployment complete!"
    echo "Start: sudo systemctl start zlmediakit"
    echo "Logs:  journalctl -u zlmediakit"
    ;;
  *)
    echo "Unsupported OS. Please install ZLMediaKit manually."
    exit 1
    ;;
esac
