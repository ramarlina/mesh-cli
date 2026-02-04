#!/usr/bin/env node

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");

const VERSION = process.env.MSH_VERSION || "latest";
const REPO = "ramarlina/mesh-cli";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    https.get(
      `https://api.github.com/repos/${REPO}/releases/latest`,
      { headers: { "User-Agent": "mesh-cli" } },
      (res) => {
        if (res.statusCode !== 200) {
          reject(new Error(`GitHub API returned ${res.statusCode}. Ensure the repo '${REPO}' has at least one release.`));
          return;
        }
        let data = "";
        res.on("data", (chunk) => (data += chunk));
        res.on("end", () => {
          try {
            const json = JSON.parse(data);
            if (!json.tag_name) {
              reject(new Error("No release tag found. Please create a release on GitHub first."));
              return;
            }
            resolve(json.tag_name);
          } catch (e) {
            reject(e);
          }
        });
      }
    ).on("error", reject);
  });
}

async function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, { headers: { "User-Agent": "mesh-cli" } }, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        download(res.headers.location, dest).then(resolve).catch(reject);
        return;
      }
      res.pipe(file);
      file.on("finish", () => {
        file.close();
        resolve();
      });
    }).on("error", (err) => {
      fs.unlink(dest, () => { });
      reject(err);
    });
  });
}

async function extract(archive, dest) {
  const isWindows = process.platform === "win32";
  if (isWindows) {
    execSync(`tar -xzf "${archive}" -C "${path.dirname(dest)}"`, { stdio: "ignore" });
  } else {
    execSync(`tar -xzf "${archive}" -C "${path.dirname(dest)}"`, { stdio: "ignore" });
  }
}

async function main() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    console.error(`Unsupported platform: ${process.platform} ${process.arch}`);
    process.exit(1);
  }

  const binDir = path.join(__dirname, "..", "bin");
  const binPath = path.join(binDir, process.platform === "win32" ? "mesh.exe" : "mesh");

  // Skip if binary already exists and is non-empty (for CI caching)
  if (fs.existsSync(binPath) && fs.statSync(binPath).size > 0) {
    console.log("mesh binary already exists, skipping download");
    return;
  }

  console.log("Installing mesh...");

  try {
    const version = VERSION === "latest" ? await getLatestVersion() : VERSION;
    if (!version) {
      throw new Error("Could not determine version. Please ensure the repository has a release.");
    }

    const ext = platform === "windows" ? "zip" : "tar.gz";
    const filename = `mesh_${version.replace("v", "")}_${platform}_${arch}.${ext}`;
    const url = `https://github.com/${REPO}/releases/download/${version}/${filename}`;

    const tmpDir = path.join(__dirname, "..", ".tmp");
    fs.mkdirSync(tmpDir, { recursive: true });
    fs.mkdirSync(binDir, { recursive: true });

    const archivePath = path.join(tmpDir, filename);

    console.log(`Downloading ${filename}...`);
    await download(url, archivePath);

    console.log("Extracting...");
    await extract(archivePath, binPath);

    // Move binary to bin directory
    const extractedBin = path.join(tmpDir, process.platform === "win32" ? "mesh.exe" : "mesh");
    if (fs.existsSync(extractedBin)) {
      fs.renameSync(extractedBin, binPath);
    }

    // Make executable
    if (process.platform !== "win32") {
      fs.chmodSync(binPath, 0o755);
    }

    // Cleanup
    fs.rmSync(tmpDir, { recursive: true, force: true });

    console.log("mesh installed successfully!");
  } catch (err) {
    console.error("Failed to install mesh:", err.message);
    process.exit(1);
  }
}

main();
