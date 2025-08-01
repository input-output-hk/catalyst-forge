const core = require("@actions/core");
const github = require("@actions/github");
const tc = require("@actions/tool-cache");
const cache = require("@actions/cache");
const fs = require("fs");
const path = require("path");

const assetPrefix = "forge-cli";
const releaseName = "forge-cli";
const repoOwner = "input-output-hk";
const repoName = "catalyst-forge";

async function run() {
  try {
    const githubToken = core.getInput("github_token");
    const version = core.getInput("version");
    const enableCaching = core.getBooleanInput("enable_caching");

    const octokit = github.getOctokit(githubToken);

    if (version !== "latest" && !isSemVer(version)) {
      core.setFailed(`Invalid version: ${version}`);
      return;
    }

    let assetUrl;
    let actualVersion = version;

    if (version === "latest") {
      assetUrl = await getLatestAsset(octokit);
      // For "latest", we need to determine the actual version for caching
      actualVersion = await getLatestVersion(octokit);
    } else {
      assetUrl = await getVersionedAsset(octokit, version);
    }

    // Try to restore from GitHub cache if caching is enabled
    if (enableCaching) {
      const cacheKey = `forge-cli-${actualVersion}-${process.platform}-${process.arch}`;
      const cachePath = path.join(
        process.cwd(),
        ".forge-cache",
        releaseName,
        actualVersion,
        process.arch,
      );

      core.info(
        `Attempting to restore from GitHub cache with key: ${cacheKey}`,
      );
      const cacheHit = await cache.restoreCache([cachePath], cacheKey);
      if (cacheHit) {
        core.info(`Restored cached version ${actualVersion} to ${cachePath}`);
        core.addPath(cachePath);
        return;
      }
    }

    core.info(`Downloading version ${actualVersion} from ${assetUrl}`);
    const downloadPath = await tc.downloadTool(
      assetUrl,
      undefined,
      `token ${githubToken}`,
      {
        accept: "application/octet-stream",
      },
    );

    if (enableCaching) {
      const cacheKey = `forge-cli-${actualVersion}-${process.platform}-${process.arch}`;
      const cachePath = path.join(
        process.cwd(),
        ".forge-cache",
        releaseName,
        actualVersion,
        process.arch,
      );

      fs.mkdirSync(cachePath, { recursive: true });
      const extractPath = await tc.extractTar(downloadPath, cachePath);

      core.info(`Caching tool with key: ${releaseName} ${actualVersion}`);
      toolPath = await tc.cacheDir(extractPath, releaseName, actualVersion);
      core.addPath(toolPath);

      core.info(`Saving to GitHub cache with key: ${cacheKey}`);
      await cache.saveCache([cachePath], cacheKey);
    } else {
      const extractPath = await tc.extractTar(downloadPath);
      toolPath = await tc.cacheDir(extractPath, releaseName, actualVersion);
      core.addPath(toolPath);
    }

    core.info(`Installed forge CLI version ${actualVersion} to ${toolPath}`);
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};

/**
 * Returns the platform suffix for the current platform.
 * @returns {string} The platform suffix for the current platform.
 */
function getAssetName() {
  const platform = `${process.platform}/${process.arch}`;
  let platformSuffix;
  switch (platform) {
    case "linux/x64":
      platformSuffix = "linux-amd64";
      break;
    case "linux/arm64":
      platformSuffix = "linux-arm64";
      break;
    case "darwin/x64":
      platformSuffix = "darwin-amd64";
      break;
    case "darwin/arm64":
      platformSuffix = "darwin-arm64";
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  return `${assetPrefix}-${platformSuffix}.tar.gz`;
}

/**
 * Returns a list of releases for the repo.
 * @param {Object} octokit The octokit instance to use.
 * @returns {Object} The release object for the latest release.
 */
async function getReleases(octokit) {
  const { data: releases } = await octokit.rest.repos.listReleases({
    owner: repoOwner,
    repo: repoName,
  });

  return releases;
}

/**
 * Returns the download URL of the latest asset.
 * @param {Object} octokit The octokit instance to use.
 * @returns {Promise<string>} The download URL of the latest asset.
 */
async function getLatestAsset(octokit) {
  const assetName = getAssetName();
  const releases = await getReleases(octokit);

  for (let i = 0; i < releases.length; i++) {
    const asset = releases[i].assets.find((a) => a.name === assetName);

    if (asset) {
      return asset.url;
    }
  }

  throw new Error(`No asset found for ${assetName}`);
}

/**
 * Returns the version number of the latest release.
 * @param {Object} octokit The octokit instance to use.
 * @returns {Promise<string>} The version number of the latest release.
 */
async function getLatestVersion(octokit) {
  const releases = await getReleases(octokit);

  for (let i = 0; i < releases.length; i++) {
    const release = releases[i];
    const asset = release.assets.find((a) => a.name === getAssetName());

    if (asset) {
      // Extract version from tag name (e.g., "forge-cli/v1.2.3" -> "1.2.3")
      const versionMatch = release.tag_name.match(
        new RegExp(`${releaseName}/v(.+)`),
      );
      if (versionMatch) {
        return versionMatch[1];
      }
    }
  }

  throw new Error("No latest version found");
}

/**
 * Returns the download URL of the asset for the given version.
 * @param {Object} octokit The octokit instance to use.
 * @param {string} version The version of the asset to get.
 * @returns {Promise<string>} The download URL of the asset.
 */
async function getVersionedAsset(octokit, version) {
  const assetName = getAssetName();
  const releases = await getReleases(octokit);

  const targetRelease = releases.find(
    (r) => r.tag_name === `${releaseName}/v${version}`,
  );
  if (!targetRelease) {
    throw new Error(`Version ${version} not found`);
  }

  const asset = targetRelease.assets.find((a) => a.name === assetName);
  if (!asset) {
    throw new Error(`No asset found for ${assetName}`);
  }

  return asset.url;
}

/**
 * Checks if the given version is a valid semantic version.
 * @param {string} version The version to check.
 * @returns {bool} True if the version is a valid semantic version, false otherwise.
 */
function isSemVer(version) {
  return /^\d+\.\d+\.\d+$/.test(version);
}
