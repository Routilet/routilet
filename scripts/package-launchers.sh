#!/usr/bin/env bash
set -euo pipefail
repo="$(cd -- "$(dirname -- "$0")/.." && pwd)"
out="$(mkdir -p "$1" && cd -- "$1" && pwd)"
version="$(cat "$repo/launcher/VERSION")"
# Guide metadata only, not a dependency on Companion source or package.json.
companion_version=0.2.8
os="$(go env GOOS)"
arch="$(go env GOARCH)"
case "$os-$arch" in windows-amd64|darwin-arm64) ;; *) echo "Unsupported phase-01 build target: $os-$arch" >&2; exit 1;; esac
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]] || exit 1
bundle="$out/routilet-phase01-$version-$os-$arch"
mkdir -p "$bundle/images"
to_shell_path() {
  if command -v cygpath >/dev/null 2>&1; then cygpath -u "$1"; else printf '%s\n' "$1"; fi
}
(
  cd "$repo/launcher"
  go mod download
  printf 'Go runtime\n\n'
  cat "$(to_shell_path "$(go env GOROOT)")/LICENSE"
  go list -m -f '{{if not .Main}}{{.Path}}{{"\t"}}{{.Dir}}{{end}}' all |
    while IFS=$'\t' read -r module directory; do
      [[ -n "$directory" ]] || continue
      directory="$(to_shell_path "$directory")"
      for file in "$directory"/LICENSE* "$directory"/COPYING* "$directory"/NOTICE*; do
        [[ -f "$file" ]] || continue
        printf '\n\n%s · %s\n\n' "$module" "$(basename "$file")"
        cat "$file"
      done
    done
) > "$bundle/THIRD_PARTY_NOTICES.txt"
node "$repo/scripts/render-launcher-guide.mjs" "$repo/docs/launcher/使用说明.html" "$bundle/使用说明.html" "$companion_version" "$version" 1
cp "$repo/docs/launcher/编译环境配置.html" "$bundle/"
cp "$repo/docs/launcher/images/contest-list.png" "$bundle/images/"
grep -Fq "Accoding Companion $companion_version" "$bundle/使用说明.html"
cp "$repo/LICENSE" "$bundle/"
printf 'Phase 01 test artifact only. Use a dedicated OS test account; the launcher retains the old installation paths. Not a release.\n' > "$bundle/TEST-ONLY.txt"
if [[ "$os" == windows ]]; then
  binary="$bundle/Routilet.exe"
else
  mkdir -p "$bundle/support"
  binary="$bundle/support/accoding-launcher"
  cp "$repo/launcher/macos.command" "$bundle/Routilet.command"
  chmod +x "$bundle/Routilet.command"
fi
(cd "$repo/launcher" && CGO_ENABLED=0 go build -mod=readonly -ldflags "-X main.launcherVersion=$version -X main.managedV2Default=1" -buildvcs=false -trimpath -o "$binary" .)
if [[ "$os" == darwin ]]; then
  codesign --force --sign - "$binary"
  codesign --verify --verbose "$binary"
  # Keep executable modes when downloading the Actions artifact.
  tar -czf "$out/routilet-phase01-$version-$os-$arch.tar.gz" -C "$out" "$(basename "$bundle")"
fi
