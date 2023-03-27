#!/usr/bin/env node

"use strict";
const _typeof = typeof Symbol === "function" && typeof Symbol.iterator === "symbol" ? function (obj) { return typeof obj; } : function (obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; };

const path = require('path'),
    fs = require('fs');

// Mapping from Node's `process.arch` to Golang's `$GOARCH`
const ARCH_MAPPING = {
    "ia32": "386",
    "x64": "amd64_v1",
    "arm": "arm"
};

// Mapping between Node's `process.platform` to Golang's
const PLATFORM_MAPPING = {
    "darwin": "darwin",
    "linux": "linux",
    "win32": "windows",
    "freebsd": "freebsd"
};

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
    if (!(process.arch in ARCH_MAPPING)) {
        console.error("Installation is not supported for this architecture: " + process.arch);
        return;
    }

    if (!(process.platform in PLATFORM_MAPPING)) {
        console.error("Installation is not supported for this platform: " + process.platform);
        return;
    }

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

/**
 * Reads the configuration from application's package.json,
 * validates properties, copies the binary from the package and stores at
 * ./bin in the package's root. NPM already has support to install binary files
 * specific locations when invoked with "npm install -g"
 *
 *  See: https://docs.npmjs.com/files/package.json#bin
 */

function install(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    const source = path.join(__dirname, 'dist', `enonic_${PLATFORM_MAPPING[process.platform]}_${ARCH_MAPPING[process.arch]}`, opts.binName);

    if (!fs.existsSync(source)) {
        console.error('Downloaded binary does not contain the binary specified in configuration - ' + opts.binName);
        return;
    }

    const targetPath = path.join(__dirname, opts.binPath);
    const target = path.join(targetPath, opts.binName);

    console.log(`Copying the relevant binary for your platform ${process.platform}`);
    fs.copyFileSync(source, target, fs.constants.COPYFILE_FICLONE);
}

function preinstall(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    const targetPath = path.join(__dirname, opts.binPath);
    const unusedBinary = path.join(targetPath, opts.binNameUnused);

    console.log(`Deleting unused binary: ${unusedBinary}`);

    try {
        fs.unlinkSync(unusedBinary);
    } catch(err) {
        //
    }
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

    console.info('Executing prepublish script...');
    const targetPath = path.join(__dirname, opts.binPath);
    fs.mkdirSync(targetPath);

    const file1 = path.join(targetPath, opts.binName);
    const file2 = path.join(targetPath, opts.binNameUnused);

    fs.writeFileSync( file1, '' );
    fs.writeFileSync( file2, '' );

    setReleaseVersion();
}

// Parse command line arguments and call the right method
const actions = {
    "install": install,
    "preinstall": preinstall,
    "prepublish": prepublish
};

const argv = process.argv;
if (argv && argv.length > 2) {
    const cmd = process.argv[2];
    if (!actions[cmd]) {
        console.log("Invalid command. `preinstall`, `install` and `prepublish` are the only supported commands");
        process.exit(1);
    }

    actions[cmd](function (err) {
        if (err) {
            console.error(err);
            process.exit(1);
        } else {
            process.exit(0);
        }
    });
}
