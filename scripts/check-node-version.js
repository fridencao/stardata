#!/usr/bin/env node
// Guards the i18n build against Node versions that break paraglide-js.
//
// On Node 18 the compiler fails with "ReferenceError: crypto is not defined"
// because it relies on the global WebCrypto API, which only became global in
// Node 19+. The failure surfaces as an opaque paraglide error, and because the
// generated messages simply stay stale, a build can silently ship outdated
// translations. Fail loudly and early instead.
//
// See .nvmrc (Node 22) for the version this repo is developed and released on.

const MIN_MAJOR = 20;

const major = Number(process.versions.node.split(".")[0]);

if (Number.isNaN(major) || major < MIN_MAJOR) {
  console.error(
    [
      "",
      `✗ Node ${process.versions.node} is too old to compile the i18n messages.`,
      `  paraglide-js needs the global WebCrypto API, available from Node ${MIN_MAJOR}.`,
      "  On Node 18 it fails with: ReferenceError: crypto is not defined",
      "",
      "  This repo targets the version in .nvmrc:",
      "    nvm use        # or: nvm install 22 && nvm use 22",
      "",
    ].join("\n"),
  );
  process.exit(1);
}
