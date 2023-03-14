#!/usr/bin/env node

"use strict";

const path = require('path');
const fs = require('fs');
const helper = require("./helper");

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


function parsePackageJson() {
    if (!(process.arch in ARCH_MAPPING)) {
        console.error("Installation is not supported for this architecture: " + process.arch);
        return;
    }

    if (!(process.platform in PLATFORM_MAPPING)) {
        console.error("Installation is not supported for this platform: " + process.platform);
        return;
    }

    return helper.parsePackageJson();
}

function postinstall(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    const source = path.join(dirName, 'dist', `${opts.project}_${PLATFORM_MAPPING[process.platform]}_${ARCH_MAPPING[process.arch]}`, opts.binName);

    if (!fs.existsSync(source)) {
        console.error(`Looking for ${opts.binName} at "${source}": Not found.`);
        return;
    }

    const target = path.join(dirName, opts.binPath, opts.binName);

    console.info(`Copying opts.binName from "${source}" to "${target}"`);
    fs.copyFileSync(source, target, fs.constants.COPYFILE_FICLONE);

    if (process.platform === "win32") {
        postprocessBinaries(opts.project);
    }
}

function postprocessBinaries(project) {
    const { execSync } = require("child_process");
    const npmRoot = execSync("npm config get prefix").toString().trim();
    const files = fs.readdirSync(npmRoot);

    files.forEach(file => {
        const extension = path.extname(file);
        const fileName = path.basename(file, extension);

        if (fileName.startsWith(project) && fileName !== project) {
            console.info(`Renaming "${file}" to "${project}${extension}"`);
            fs.renameSync(path.join(npmRoot, file), path.join(npmRoot, `${project}${extension}`));
        }
    })

    cleanWindowsBinary(project);
}

function cleanWindowsBinary(project) {
    console.info(`Deleting Windows binary...`);
    const { execSync } = require("child_process");
    const npmRoot = execSync("npm config get prefix").toString().trim();

    try {
        fs.unlinkSync(path.join(npmRoot, `${project}.exe`));
    } catch(err) {
        //
    }
}

function cleanUnusedBinary(opts) {
    const unusedBinary = path.join(dirName, opts.binPath, opts.binNameUnused);
    console.info(`Deleting unused binary: ${unusedBinary}`);

    try {
        fs.unlinkSync(unusedBinary);
    } catch(err) {
        //
    }
}

function preinstall(callback) {
    const opts = parsePackageJson();
    if (!opts) {
        return callback("Invalid inputs");
    }

    cleanUnusedBinary(opts);
    cleanWindowsBinary(opts.project);
}

// Parse command line arguments and call the right method
const actions = {
    postinstall,
    preinstall
};

const argv = process.argv;
if (argv && argv.length > 2) {
    const cmd = process.argv[2];
    if (!actions[cmd]) {
        console.info(`Invalid command: ${cmd}. "preinstall" and "postinstall" are the only supported commands`);
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
