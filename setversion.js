#!/usr/bin/env node

"use strict";

const { execSync } = require("child_process");
const version = execSync("git describe --tags --abbrev=0").toString().slice(1).trim();

execSync(`awk -v ver="${version}" '{gsub(/"version": ".*"/,"\\"version\\": \\""ver"\\"")}1' package.json > tmp && mv tmp package.json`);
