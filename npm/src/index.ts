import { spawn } from "node:child_process";
import crypto from "node:crypto";
import fs from "node:fs";
import https from "node:https";
import os from "node:os";
import path from "node:path";

const CLI_VERSION = "2.0.0";
const GITHUB_REPO = process.env.NAAGMANI_DIST_REPO || "bhakha-services/naagmani-cli";
const RELEASE_BASE_URL = process.env.NAAGMANI_RELEASE_BASE_URL || `https://github.com/${GITHUB_REPO}/releases/download/v${CLI_VERSION}`;

interface TargetPlatform {
  os: "windows" | "linux" | "darwin";
  arch: "amd64" | "arm64";
  ext: string;
  name: string;
}

/**
 * Detect current platform and architecture.
 */
function getTargetPlatform(): TargetPlatform {
  const platform = process.platform;
  const arch = process.arch;

  let osName: "windows" | "linux" | "darwin";
  let ext = "";

  if (platform === "win32") {
    osName = "windows";
    ext = ".exe";
  } else if (platform === "linux") {
    osName = "linux";
  } else if (platform === "darwin") {
    osName = "darwin";
  } else {
    console.error(`\x1b[31m✖ Error: Unsupported operating system: ${platform}\x1b[0m`);
    process.exit(1);
  }

  let archName: "amd64" | "arm64";
  if (arch === "x64") {
    archName = "amd64";
  } else if (arch === "arm64") {
    archName = "arm64";
  } else {
    console.error(`\x1b[31m✖ Error: Unsupported architecture: ${arch}. Naagmani supports x64 and arm64.\x1b[0m`);
    process.exit(1);
  }

  return {
    os: osName,
    arch: archName,
    ext,
    name: `naagmani-${osName}-${archName}${ext}`,
  };
}

/**
 * Download a file via HTTPS with 301/302 redirect handling.
 */
function downloadFile(url: string, destPath: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const followRedirect = (currentUrl: string, maxRedirects = 5) => {
      if (maxRedirects <= 0) {
        return reject(new Error("Too many HTTP redirects while downloading CLI binary"));
      }

      https
        .get(currentUrl, (res) => {
          if (res.statusCode && res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
            return followRedirect(res.headers.location, maxRedirects - 1);
          }

          if (res.statusCode !== 200) {
            return reject(new Error(`Failed to download binary: HTTP ${res.statusCode} ${res.statusMessage || ""}`));
          }

          const fileStream = fs.createWriteStream(destPath);
          res.pipe(fileStream);

          fileStream.on("finish", () => {
            fileStream.close();
            resolve();
          });

          fileStream.on("error", (err) => {
            fs.unlink(destPath, () => {});
            reject(err);
          });
        })
        .on("error", (err) => {
          fs.unlink(destPath, () => {});
          reject(err);
        });
    };

    followRedirect(url);
  });
}

/**
 * Fetch release checksums if published.
 */
async function fetchReleaseChecksums(version: string): Promise<Map<string, string>> {
  const checksumUrl = `${RELEASE_BASE_URL}/checksums.txt`;
  return new Promise((resolve) => {
    const checksums = new Map<string, string>();
    https
      .get(checksumUrl, (res) => {
        if (res.statusCode !== 200) {
          return resolve(checksums); // Checksum file optional/not available
        }
        let data = "";
        res.on("data", (chunk) => {
          data += chunk;
        });
        res.on("end", () => {
          const lines = data.split("\n");
          for (const line of lines) {
            const parts = line.trim().split(/\s+/);
            if (parts.length >= 2) {
              const hash = parts[0];
              const filename = parts[1].replace(/^\*/, "");
              checksums.set(filename, hash);
            }
          }
          resolve(checksums);
        });
      })
      .on("error", () => {
        resolve(checksums);
      });
  });
}

/**
 * Calculate SHA-256 of local file.
 */
function calculateSHA256(filePath: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash("sha256");
    const stream = fs.createReadStream(filePath);
    stream.on("data", (chunk) => hash.update(chunk));
    stream.on("end", () => resolve(hash.digest("hex")));
    stream.on("error", (err) => reject(err));
  });
}

/**
 * Resolve or download the native Go CLI binary.
 */
export async function resolveBinary(): Promise<string> {
  const target = getTargetPlatform();

  // 1. Explicit override via NAAGMANI_CLI_PATH
  if (process.env.NAAGMANI_CLI_PATH && fs.existsSync(process.env.NAAGMANI_CLI_PATH)) {
    return process.env.NAAGMANI_CLI_PATH;
  }

  // 2. Development monorepo sibling path
  try {
    const baseDir = typeof __dirname !== "undefined" ? __dirname : path.resolve();
    const monorepoCli = path.resolve(
      baseDir,
      "..",
      "..",
      "..",
      "tools",
      "naagmani-cli",
      target.os === "windows" ? "naagmani.exe" : "naagmani"
    );
    if (fs.existsSync(monorepoCli)) {
      return monorepoCli;
    }
  } catch {
    // Ignore resolution error
  }

  // 3. User Cache directory (~/.naagmani/bin/)
  const homeDir = os.homedir();
  const cacheDir = path.join(homeDir, ".naagmani", "bin");
  const cachedBinary = path.join(cacheDir, `naagmani-v${CLI_VERSION}-${target.os}-${target.arch}${target.ext}`);

  if (fs.existsSync(cachedBinary)) {
    return cachedBinary;
  }

  // 4. Bundled binary inside package
  const bundledPath = path.resolve(__dirname, "..", "bin", target.name);
  if (fs.existsSync(bundledPath)) {
    return bundledPath;
  }

  // 5. Download from official GitHub Release
  if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true });
  }

  const downloadUrl = `${RELEASE_BASE_URL}/${target.name}`;
  const tempDownload = path.join(cacheDir, `download-${Date.now()}-${target.name}`);

  console.log(`\x1b[34mℹ Naagmani CLI native engine not found. Downloading v${CLI_VERSION} for ${target.os}/${target.arch}...\x1b[0m`);

  try {
    await downloadFile(downloadUrl, tempDownload);

    // Verify checksum if available
    const checksums = await fetchReleaseChecksums(CLI_VERSION);
    if (checksums.has(target.name)) {
      const expectedHash = checksums.get(target.name);
      const actualHash = await calculateSHA256(tempDownload);
      if (expectedHash !== actualHash) {
        fs.unlinkSync(tempDownload);
        throw new Error(`Checksum mismatch for ${target.name}. Expected ${expectedHash}, got ${actualHash}`);
      }
    }

    fs.chmodSync(tempDownload, 0o755);
    fs.renameSync(tempDownload, cachedBinary);
    console.log(`\x1b[32m✔ Naagmani CLI v${CLI_VERSION} installed successfully to ${cachedBinary}\x1b[0m\n`);
    return cachedBinary;
  } catch (err: any) {
    if (fs.existsSync(tempDownload)) {
      try {
        fs.unlinkSync(tempDownload);
      } catch {}
    }
    console.error(`\x1b[31m✖ Error downloading Naagmani binary: ${err.message}\x1b[0m`);
    console.error(`Please install manually or download from: https://github.com/${GITHUB_REPO}/releases/tag/v${CLI_VERSION}`);
    process.exit(4); // Deterministic Network/Cloud Failure Exit Code
  }
}

/**
 * Entry point: executes the resolved Go binary, forwards stdio and exit code.
 */
export async function main(): Promise<void> {
  const binaryPath = await resolveBinary();
  const args = process.argv.slice(2);

  const child = spawn(binaryPath, args, {
    stdio: "inherit",
    env: process.env,
  });

  // Forward termination signals to child process
  const signals: NodeJS.Signals[] = ["SIGINT", "SIGTERM", "SIGHUP"];
  signals.forEach((sig) => {
    process.on(sig, () => {
      if (!child.killed) {
        child.kill(sig);
      }
    });
  });

  child.on("error", (err) => {
    console.error(`\x1b[31m✖ Execution error: ${err.message}\x1b[0m`);
    process.exit(1);
  });

  child.on("close", (code) => {
    process.exit(code ?? 0);
  });
}

// Auto-run if executed as main entry
main().catch((err) => {
  console.error(`\x1b[31m✖ Unexpected error: ${err.message}\x1b[0m`);
  process.exit(1);
});
