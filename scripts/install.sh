#!/bin/sh
# Install an official lazygit-ai release without Go or administrator privileges.
set -eu

main() {
    repo=javiermoya2504/lazygit-ai
    install_dir=${LAZYGIT_AI_INSTALL_DIR:-"$HOME/.local/bin"}
    for tool in curl tar awk; do
        command -v "$tool" >/dev/null 2>&1 || { echo "Required command: $tool" >&2; exit 1; }
    done
    case "$(uname -s)" in
        Darwin) os=darwin ;;
        Linux) os=linux ;;
        *) echo 'Supported systems: macOS and Linux. Use install.ps1 on Windows.' >&2; exit 1 ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch=x86_64 ;;
        aarch64|arm64) arch=arm64 ;;
        armv6l|armv7l) arch=armv6 ;;
        i386|i686) arch=32-bit ;;
        *) echo 'Unsupported CPU architecture.' >&2; exit 1 ;;
    esac
    tag=${LAZYGIT_AI_VERSION:-}
    if [ -z "$tag" ]; then
        release_url=$(curl -fsSL --proto '=https' --proto-redir '=https' -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest") || {
            echo "Could not find the latest release: https://github.com/$repo/releases" >&2; exit 1;
        }
        tag=${release_url##*/}
    fi
    # Keep URL components and archive names restricted to release version syntax.
    printf '%s\n' "$tag" | awk '/^v[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$/ { ok=1 } END { exit !ok }' || {
        echo "Invalid release version: $tag (expected vX.Y.Z)" >&2; exit 1;
    }
    archive="lazygit-ai_${tag#v}_${os}_${arch}.tar.gz"
    base="https://github.com/$repo/releases/download/$tag"
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT
    trap 'exit 1' HUP INT TERM
    echo "Downloading lazygit-ai $tag ($os/$arch)..."
    curl -fsSL --proto '=https' --proto-redir '=https' "$base/$archive" -o "$tmp_dir/$archive"
    curl -fsSL --proto '=https' --proto-redir '=https' "$base/checksums.txt" -o "$tmp_dir/checksums.txt"
    expected=$(awk -v file="$archive" '$2 == file { print $1 }' "$tmp_dir/checksums.txt")
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tmp_dir/$archive" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$tmp_dir/$archive" | awk '{print $1}')
    else
        echo 'Install sha256sum or shasum to verify the download.' >&2; exit 1
    fi
    [ -n "$expected" ] && [ "$actual" = "$expected" ] || { echo 'SHA256 verification failed; installation cancelled.' >&2; exit 1; }
    tar -xzf "$tmp_dir/$archive" -C "$tmp_dir" lazygit-ai
    mkdir -p "$install_dir"
    install -m 755 "$tmp_dir/lazygit-ai" "$install_dir/lazygit-ai"
    echo "Installed: $install_dir/lazygit-ai"
    case ":$PATH:" in
        *":$install_dir:"*) ;;
        *) printf 'Add this directory to PATH in your shell configuration: %s\n' "$install_dir" ;;
    esac
    if ! command -v git >/dev/null 2>&1; then echo 'Install Git before running lazygit-ai.'; fi
    echo 'Ollama setup: https://github.com/javiermoya2504/lazygit-ai#instalar-y-preparar-ollama'
}

main "$@"
