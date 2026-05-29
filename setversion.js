#!/usr/bin/env node

"use strict";

const { execSync } = require("child_process");

// Prefer the tag that triggered the release (GitHub Actions sets GITHUB_REF_NAME).
// Fall back to git describe for local/manual runs. Using git describe alone is
// unreliable when several tags point at the same commit, since it picks one arbitrarily.
const refName = process.env.GITHUB_REF_NAME;
const rawTag = refName && refName.startsWith("v")
  ? refName
  : execSync("git describe --tags --abbrev=0").toString().trim();
const version = rawTag.slice(1).trim();

execSync(`awk -v ver="${version}" '{gsub(/"version": ".*"/,"\\"version\\": \\""ver"\\"")}1' package.json > tmp && mv tmp package.json`);
