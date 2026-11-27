#!/bin/sh
set -eu
bundle_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
export ACCODING_LAUNCHER_GUIDE="$bundle_dir/使用说明.html"
exec "$bundle_dir/support/accoding-launcher"
