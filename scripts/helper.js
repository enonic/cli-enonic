#!/usr/bin/env node

"use strict";
const _typeof = typeof Symbol === "function" && typeof Symbol.iterator === "symbol" ? function (obj) { return typeof obj; } : function (obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; };

const path = require('path');
const fs = require('fs');

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

module.exports = {
    parsePackageJson: () => {
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

        const project = packageJson.goBinary;
        let binName = project;
        let binNameUnused = binName + ".exe";
        const binPath = packageJson.directories.bin;

        // Binary name on Windows has .exe suffix
        if (process.platform === "win32") {
            binNameUnused = binName;
            binName += ".exe";
        }

        return {
            project,
            binName,
            binNameUnused,
            binPath
        };
    }
}
