#!/bin/bash
find dist/*/ -name enonic | while read -r file; do
    echo "Running chmod 755 on $file"
    chmod 755 "$file"
done
