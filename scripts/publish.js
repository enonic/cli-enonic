#!/usr/bin/env node

"use strict";
const _typeof = typeof Symbol === "function" && typeof Symbol.iterator === "symbol" ? function (obj) { return typeof obj; } : function (obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; };

const path = require('path'),
    fs = require('fs');

function validateConfiguration(packageJson) {
    if (!packageJson.directories || _typeof(packageJson.directories) !== "object") {
        return "'directories' property must be defined and be an object";
    }

    if (!packageJson.goBinary) {
        return "'goBinary' property is required";
    }

    if (!packageJson.directories.bin) {
        return "'directories.bin' property is required";
    }
}

function parsePackageJson() {
    const packageJsonPath = path.join(".", "package.json");
    if (!fs.existsSync(packageJsonPath)) {
        console.error("Unable to find package.json. " + "Please run this script at root of the package you want to be installed");
        return;
    }

    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath));
    const error = validateConfiguration(packageJson);
    if (error && error.length > 0) {
        console.error("Invalid package.json: " + error);
        return;
    }

    // We have validated the config. It exists in all its glory
    let binName = packageJson.goBinary;
    let binNameUnused = binName + ".exe";
    const binPath = packageJson.directories.bin;

    // Binary name on Windows has .exe suffix
    if (process.platform === "win32") {
        binNameUnused = binName;
        binName += ".exe";
    }

    return {
        binName,
        binNameUnused,
        binPath
    };
}

function setReleaseVersion() {
    const { execSync } = require("child_process");
    const version = execSync("git describe --tags --abbrev=0").toString().slice(1).trim();

    execSync(`awk -v ver="${version}" '{gsub(/"version": ".*"/,"\\"version\\": \\""ver"\\"")}1' package.json > tmp && mv tmp package.json`);
}

function prepublish(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    console.log('Executing prepublish script...');
    const targetPath = path.join(process.cwd(), opts.binPath);

    if (!fs.existsSync(targetPath)) {
        fs.mkdirSync(targetPath);
    }

    const file1 = path.join(targetPath, opts.binName);
    const file2 = path.join(targetPath, opts.binNameUnused);

    fs.writeFileSync( file1, '' );
    fs.writeFileSync( file2, '' );

    setReleaseVersion();
}

const argv = process.argv;
if (argv && argv.length > 2) {
    const cmd = process.argv[2];
    if (cmd !== 'prepublish') {
        console.log("Invalid command. `prepublish` is the only supported command.");
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
