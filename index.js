#!/usr/bin/env node

'use strict';

const path = require('path');
const fs = require('fs');
const { arch, platform, argv, env } = require('node:process');

// Mapping from Node's `process.arch` to Golang's `$GOARCH`
const ARCH_MAPPING = {
    'ia32': '386',
    'x64': 'amd64_v1',
    'arm': 'arm_6',
    'arm64': 'arm64_v8.0'
};

// Mapping between Node's `process.platform` to Golang's
const PLATFORM_MAPPING = {
    'darwin': 'darwin',
    'linux': 'linux',
    'win32': 'windows',
    'freebsd': 'freebsd'
};

const validateConfiguration = (packageJson) => {
    if (!packageJson.goBinary) {
        return '"goBinary" property is required in package.json';
    }

    return '';
}

const parsePackageJson = () => {
    const packageJsonPath = path.resolve(__dirname, 'package.json');
    if (!fs.existsSync(packageJsonPath)) {
        console.error('Unable to find package.json. Please run this script at root of the package you want to be installed');
        process.exit(1);
    }

    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath));
    const error = validateConfiguration(packageJson);
    if (error && error.length > 0) {
        console.error(`Invalid package.json: ${error}`);
        process.exit(1);
    }

    if (!(platform in PLATFORM_MAPPING)) {
        console.error(`Installation is not supported for platform "${platform}"`);
        process.exit(1);
    }

    /*
        This part is used for cross-platform verification of binaries during CI/CD.
    */
    let thisArch = arch;
    if (env['NODE_ARCH']) {
        if (platform === 'win32' && env['NODE_ARCH'] === 'arm64') {
            console.log(`Skipping verification for ${platform} ${arch}`);
            process.exit(0);
        }
        if (platform === 'linux' && env['NODE_ARCH'] === 'arm64') {
            thisArch = 'arm';
        } else {
            thisArch = env['NODE_ARCH'];
        }
    }
    /**/

    if (!(thisArch in ARCH_MAPPING)) {
        console.error(`Installation is not supported for architecture "${thisArch}"`);
        process.exit(1);
    }

    const project = packageJson.goBinary;
    let binName = project;

    // Binary name on Windows has .exe suffix
    if (platform === 'win32') {
        binName += '.exe';
    }

    const binFolder = `${project}_${PLATFORM_MAPPING[platform]}_${ARCH_MAPPING[thisArch]}`;
    const binPath = path.resolve(__dirname, 'dist', binFolder, binName);

    return {
        project,
        binPath
    };
}

if (argv) {
    try {
        const opts = parsePackageJson();
        if (!opts) {
            console.error('Invalid package.json');
            process.exit(1);
        }

        if (!fs.existsSync(opts.binPath)) {
            console.error(`Binary not found at ${opts.binPath}`);
            process.exit(1);
        }

        const {spawn} = require('child_process');
        spawn(opts.binPath, argv.slice(2), { stdio: 'inherit' });
    }
    catch (e) {
        console.error(e);
    }
}
