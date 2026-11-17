#!/usr/bin/env bash
# wtui installer: curl -fsSL https://raw.githubusercontent.com/boutdecode/wtui/main/install.sh | bash
# Env: WTUI_VERSION (default: latest), WTUI_INSTALL_DIR, WTUI_REPO (default: boutdecode/wtui)
set -euo pipefail

main() {
  local repo="${WTUI_REPO:-boutdecode/wtui}"
  local os arch version base tmp dir sudo=""

  case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) echo "wtui: unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "wtui: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac

  command -v curl >/dev/null || { echo "wtui: curl is required" >&2; exit 1; }
  command -v tar >/dev/null || { echo "wtui: tar is required" >&2; exit 1; }

  version="${WTUI_VERSION:-}"
  if [ -z "$version" ]; then
    version="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")"
    version="${version##*/}"
  fi
  [ -n "$version" ] && [ "$version" != "latest" ] || { echo "wtui: cannot determine latest version (no release yet?)" >&2; exit 1; }

  base="https://github.com/$repo/releases/download/$version"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  local asset="wtui_${os}_${arch}.tar.gz"
  echo "Downloading wtui $version ($os/$arch)..."
  curl -fsSL "$base/$asset" -o "$tmp/$asset"
  curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

  local want got
  want="$(awk -v a="$asset" '$2 == a {print $1}' "$tmp/checksums.txt")"
  [ -n "$want" ] || { echo "wtui: no checksum for $asset" >&2; exit 1; }
  if command -v sha256sum >/dev/null; then
    got="$(sha256sum "$tmp/$asset" | awk '{print $1}')"
  else
    got="$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')"
  fi
  [ "$want" = "$got" ] || { echo "wtui: checksum mismatch" >&2; exit 1; }

  tar -xzf "$tmp/$asset" -C "$tmp" wtui

  dir="${WTUI_INSTALL_DIR:-}"
  if [ -z "$dir" ]; then
    if [ -w /usr/local/bin ]; then
      dir=/usr/local/bin
    elif command -v sudo >/dev/null && [ -t 1 ]; then
      dir=/usr/local/bin; sudo=sudo
    else
      dir="$HOME/.local/bin"
    fi
  fi
  mkdir -p "$dir" 2>/dev/null || $sudo mkdir -p "$dir"
  $sudo install -m 0755 "$tmp/wtui" "$dir/wtui"

  echo "Installed $dir/wtui"
  case ":$PATH:" in
    *":$dir:"*) ;;
    *) echo "Note: $dir is not in your PATH." ;;
  esac
}

main "$@"
