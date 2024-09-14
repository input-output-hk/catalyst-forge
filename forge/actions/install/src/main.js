const core = require("@actions/core");
const github = require("@actions/github");
const tc = require("@actions/tool-cache");

const repoOwner = "input-output-hk";
const repoName = "catalyst-forge";

async function run() {
  try {
    //const earthlyToken = core.getInput("earthly_token");
    const githubToken = core.getInput("github_token");
    const version = core.getInput("version");

    const octokit = github.getOctokit(githubToken);

    if (version !== "latest" && version !== "local" && !isSemVer(version)) {
      core.setFailed("Invalid version");
      return;
    }

    let assetUrl;
    if (version === "local") {
      await installLocal();
    } else if (version === "latest") {
      assetUrl = await getLatestAsset(octokit);
    } else {
      assetUrl = await getVersionedAsset(octokit, version);
    }

    core.info(`Downloading version ${version} from ${assetUrl}`);
    const downloadPath = await tc.downloadTool(
      assetUrl,
      undefined,
      `token ${githubToken}`,
      {
        accept: "application/octet-stream",
      },
    );
    const extractPath = await tc.extractTar(downloadPath);
    core.addPath(extractPath);

    core.info(`Installed forge version ${version} to ${extractPath}`);
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

  return `forge-${platformSuffix}.tar.gz`;
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
      return asset.browser_download_url;
    }
  }

  throw new Error(`No asset found for ${assetName}`);
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

  const targetRelease = releases.find((r) => r.tag_name === `v${version}`);
  if (!targetRelease) {
    throw new Error(`Version ${version} not found`);
  }

  const asset = targetRelease.assets.find((a) => a.name === assetName);
  if (!asset) {
    throw new Error(`No asset found for ${assetName}`);
  }

  return asset.browser_download_url;
}

async function installLocal() {}

/**
 * Checks if the given version is a valid semantic version.
 * @param {string} version The version to check.
 * @returns {bool} True if the version is a valid semantic version, false otherwise.
 */
function isSemVer(version) {
  return /^\d+\.\d+\.\d+$/.test(version);
}
