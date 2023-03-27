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

const dirName = process.cwd();

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

function postinstall(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    const source = path.join(dirName, 'dist', `enonic_${PLATFORM_MAPPING[process.platform]}_${ARCH_MAPPING[process.arch]}`, opts.binName);

    if (!fs.existsSync(source)) {
        console.error(`Looking for ${opts.binName} at "${source}": Not found.`);
        return;
    }

    const target = path.join(dirName, opts.binPath, opts.binName);

    console.log(`Copying opts.binName from "${source}" to "${target}"`);
    fs.copyFileSync(source, target, fs.constants.COPYFILE_FICLONE);
}

function preinstall(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    const unusedBinary = path.join(dirName, opts.binPath, opts.binNameUnused);

    console.log(`Deleting unused binary: ${unusedBinary}`);

    try {
        fs.unlinkSync(unusedBinary);
    } catch(err) {
        //
    }
}

// Parse command line arguments and call the right method
const actions = {
    "postinstall": postinstall,
    "preinstall": preinstall
};

const argv = process.argv;
if (argv && argv.length > 2) {
    const cmd = process.argv[2];
    if (!actions[cmd]) {
        console.log("Invalid command. `preinstall` and `postinstall` are the only supported commands");
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
