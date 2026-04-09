#!/usr/bin/env bash
#---------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
#---------------------------------------------------------------------------------------------

mkdir -p ~/bin

set -e # Terminates program immediately if any command below exits with a non-zero exit status

env=prod #Set default environment to production
for arg in "$@"
do
    case $arg in
    -e|--env)
        env="$2"
        shift # Remove argument name from processing
        shift # Remove argument value from processing
        ;;
    esac
done

if [[ "$env" != "dev" && "$env" != "ppe" && "$env" != "prod" ]]; then
    echo "env $env is not allowed, acceptable values are dev, ppe and prod"
    exit
else
    echo "Downloading the devtunnel CLI..."
fi

if [ "$(uname)" == "Darwin" ]; then
    ARCH="$(uname -m)"
    if [ ${ARCH} == "arm64" ]; then
        curl -sL -o ~/bin/devtunnel https://tunnelsassets$env.blob.core.windows.net/cli/osx-arm64-devtunnel || { echo "Cannot install CLI. Aborting"; exit 1; }
    elif [ ${ARCH} == "x86_64" ]; then
        curl -sL -o ~/bin/devtunnel https://tunnelsassets$env.blob.core.windows.net/cli/osx-x64-devtunnel || { echo "Cannot install CLI. Aborting"; exit 1; }
    else
        echo "unsupported architecture ${ARCH}"
        exit
    fi
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    ARCH="$(uname -m)"
    if [ "$(expr substr ${ARCH} 1 5)" == "arm64" ] || [ "$(expr substr ${ARCH} 1 7)" == "aarch64" ]; then
        curl -sL -o ~/bin/devtunnel https://tunnelsassets$env.blob.core.windows.net/cli/linux-arm64-devtunnel || { echo "Cannot install CLI. Aborting"; exit 1; }
    elif [ "$(expr substr ${ARCH} 1 6)" == "x86_64" ]; then
        curl -sL -o ~/bin/devtunnel https://tunnelsassets$env.blob.core.windows.net/cli/linux-x64-devtunnel || { echo "Cannot install CLI. Aborting"; exit 1; }
    else
        echo "unsupported architecture ${ARCH}"
        exit
    fi
    sudo apt-get -qq update -y
    sudo apt-get -qq install -y libsecret-1-0
fi

if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
    if [[ -e ~/.zshrc ]]; then
        echo "export PATH=$PATH:$HOME/bin" >> $HOME/.zshrc
        fileUsed="~/.zshrc"
    elif  [[ -e ~/.bashrc ]]; then
        echo "export PATH=$PATH:$HOME/bin" >> $HOME/.bashrc 
        fileUsed="~/.bashrc"
    else
        echo "export PATH=$PATH:$HOME/bin" >> $HOME/.bash_profile
        fileUsed="~/.bash_profile"
    fi
fi
chmod +x ~/bin/devtunnel

echo "devtunnel CLI installed!"
echo ""
echo "Version:"
echo "    $(~/bin/devtunnel --version)"
echo ""
echo "To get started, run:"
if [[ "$fileUsed" != "" ]]; then
    echo "    source $fileUsed"
fi
echo "    devtunnel -h"

