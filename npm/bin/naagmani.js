#!/usr/bin/env node

import("../dist/index.js").catch((err) => {
  console.error("Failed to run naagmani CLI:", err);
  process.exit(1);
});
