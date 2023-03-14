#!/usr/bin/env node

"use strict";

const path = require('path');
const fs = require('fs');
const helper = require("./helper");

function setReleaseVersion() {
    const { execSync } = require("child_process");
    const version = execSync("git describe --tags --abbrev=0").toString().slice(1).trim();

    execSync(`awk -v ver="${version}" '{gsub(/"version": ".*"/,"\\"version\\": \\""ver"\\"")}1' package.json > tmp && mv tmp package.json`);
}

function prepublish(callback) {
    const opts = helper.parsePackageJson();

    if (!opts) {
        return callback("Invalid inputs");
    }

    console.info('Executing prepublish script...');
    const targetPath = path.join(process.cwd(), opts.binPath);

    if (!fs.existsSync(targetPath)) {
        fs.mkdirSync(targetPath);
    }

    const file1 = path.join(targetPath, opts.binName);
    const file2 = path.join(targetPath, opts.binNameUnused);

    fs.writeFileSync(file1, '');
    fs.writeFileSync(file2, '');

    setReleaseVersion();
}

const argv = process.argv;
if (argv && argv.length > 2) {
    const cmd = process.argv[2];
    if (cmd !== 'prepublish') {
        console.info("Invalid command. `prepublish` is the only supported command.");
        process.exit(1);
    }

    prepublish(function (err) {
        if (err) {
            console.error(err);
            process.exit(1);
        } else {
            process.exit(0);
        }
    });
}
