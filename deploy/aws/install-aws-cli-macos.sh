#!/usr/bin/env bash
# Homebrew `awscli` can fail on macOS 26 with pyexpat errors. This installs the official AWS CLI v2 to ~/.local.
set -euo pipefail
INSTALL_DIR="${HOME}/.local/aws-cli-install"
BIN_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR" "$BIN_DIR"
cd "$INSTALL_DIR"
curl -fsSL "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o AWSCLIV2.pkg
rm -rf expanded
pkgutil --expand-full AWSCLIV2.pkg expanded
AWS_EXE="$INSTALL_DIR/expanded/aws-cli.pkg/Payload/aws-cli/aws"
ln -sf "$AWS_EXE" "$BIN_DIR/aws"
echo "Installed: $("$BIN_DIR/aws" --version)"
echo "Add to PATH (zsh): echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
