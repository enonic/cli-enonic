#!/bin/bash
find dist/*/ -name enonic | while read -r file; do
    echo "Running chmod 755 on $file"
    chmod 755 "$file"
    if [[ "$file" == *darwin* ]] && command -v codesign &>/dev/null; then
        echo "Running codesign on $file"
        codesign -s - "$file"
    fi
done
