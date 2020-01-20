#!/bin/sh

# This script is meant for quick & easy install via:
#   $ wget -qO- https://raw.githubusercontent.com/enonic/cli-enonic/master/install.sh | sh

set -ex

# Set tmp variables
tmp_dir=$(mktemp -d -t enonic-XXXXXXXXXX)
tmp_archive=$tmp_dir/enonic.tar.gz

# Download and unzip cli
wget -qO- https://api.github.com/repos/enonic/cli-enonic/releases/latest \
    | grep -o "https.*Linux_64-bit.tar.gz" \
    | xargs -I {} wget {} -O $tmp_archive
tar xvzf $tmp_archive -C $tmp_dir

# Install cli
sudo -k install $tmp_dir/enonic /usr/bin/enonic

# Clean Up
rm -r $tmp_dir