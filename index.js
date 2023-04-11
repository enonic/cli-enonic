#!/usr/bin/env node

'use strict';

const path = require('path');
const fs = require('fs');

// Mapping from Node's `process.arch` to Golang's `$GOARCH`
const ARCH_MAPPING = {
    'ia32': '386',
    'x64': 'amd64_v1',
    'arm': 'arm'
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
        throw new Error('Unable to find package.json. Please run this script at root of the package you want to be installed');
    }

    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath));
    const error = validateConfiguration(packageJson);
    if (error && error.length > 0) {
        throw new Error(`Invalid package.json: ${error}`);
    }

    if (!(process.platform in PLATFORM_MAPPING)) {
        throw new Error(`Installation is not supported for platform "${process.platform}"`);
    }

    if (!(process.arch in ARCH_MAPPING)) {
        throw new Error(`Installation is not supported for architecture "${process.arch}"`);
    }

    const project = packageJson.goBinary;
    let binName = project;

    // Binary name on Windows has .exe suffix
    if (process.platform === 'win32') {
        binName += '.exe';
    }

    const binFolder = `${project}_${PLATFORM_MAPPING[process.platform]}_${ARCH_MAPPING[process.arch]}`;
    const binPath = path.resolve(__dirname, 'dist', binFolder, binName);

    return {
        project,
        binPath
    };
}

const argv = process.argv;
if (argv) {
    try {
        const opts = parsePackageJson();
        if (!opts) {
            console.error('Invalid package.json');
            return;
        }

        const {spawn} = require('child_process');
        spawn(opts.binPath, argv.slice(2), { stdio: 'inherit' });
    }
    catch (e) {
        console.error(e);
    }
}
