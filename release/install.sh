#!/bin/sh
# Based on Deno installer: Copyright 2019 the Deno authors. All rights reserved. MIT license.
# TODO(everyone): Keep this script simple and easily auditable.

set -e

os=$(uname | tr "[:upper:]" "[:lower:]")
arch=$(uname -m)
version=${1:-latest}

nifectl_uri=$(curl -s https://api.nife.io/release/$os/$arch/$version)
if [ ! "$nifectl_uri" ]; then
	echo "Error: Unable to find a nifectl release for $os/$arch/$version - see github.com/nifetency/nifectl_releases/releases for all versions" 1>&2
	exit 1
fi

nifectl_install="$HOME/.nife"

bin_dir="$nifectl_install/bin"
exe="$bin_dir/nifectl"
simexe="$bin_dir/nife"

#mkdir -p "$bin_dir"

if [ ! -d "$bin_dir" ]; then
 	mkdir -p "$bin_dir"
fi

cd $bin_dir
wget $nifectl_uri
tar -xvf *.tar.gz 
chmod +x "$exe"
#rm "$exe.tar.gz"
cd $bin_dir
#simexe="nife"

rm -rf *tar.gz

ln -sf $exe $simexe

if [ "${1}" = "prerel" ] || [ "${1}" = "pre" ]; then
	"$exe" version -s "shell-prerel"
else
	"$exe" version -s "shell"
fi

echo "nifectl was installed successfully to $exe"
if command -v nifectl >/dev/null; then
	echo "Run ‘nifectl --help' to get started"
else
	case $SHELL in
	/bin/zsh) shell_profile=".zshrc" ;;
	*) shell_profile=".bash_profile" ;;
	esac
	echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
	echo "  export NIFECTL_INSTALL=\"$nifectl_install\""
	echo "  export PATH=\"\$NIFECTL_INSTALL/bin:\$PATH\""
	echo "Run nifectl  --help to get started"
fi


