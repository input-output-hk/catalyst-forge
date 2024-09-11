const core = require("@actions/core");
const exec = require("@actions/exec");
const fs = require("fs");
const github = require("@actions/github");

async function run() {
  try {
    const nativePlatform = core.getInput("native_platform");
    const project = core.getInput("project");
    const path = core.getInput("path");
    const target = core.getInput("target");
    const token = core.getInput("github_token");

    const blueprint = await getBlueprint(project);
    const targetConfig = blueprint.project?.ci?.targets?.[target];
    let platforms = [];
    if (
      targetConfig !== undefined &&
      targetConfig.platforms !== undefined &&
      targetConfig.platforms.length > 0
    ) {
      core.info(
        `Detected multi-platform build for platforms: ${targetConfig.platforms.join(", ")}`,
      );
      platforms = targetConfig.platforms;
    } else {
      platforms = [nativePlatform];
    }

    for (const platform of platforms) {
      core.info(`Validating artifacts for platform ${platform}`);
      const platformPath = `${path}/${platform}`;
      if (!fs.existsSync(platformPath)) {
        core.setFailed(
          `Unable to find output folder for platform: ${platform}`,
        );
        return;
      }

      const files = fs.readdirSync(platformPath);
      if (files.length === 0) {
        core.setFailed(`No artifacts found for platform: ${platform}`);
        return;
      }
    }

    const result = await getTags(project);
    if (result.git === "") {
      core.info("No Git tag detected");
      return;
    }

    const assets = [];
    const gitTag = result.git;
    for (const platform of platforms) {
      let archiveName = "";
      if (gitTag.split("/").length > 1) {
        const prefix = gitTag
          .split("/")
          .slice(0, -1)
          .join("/")
          .replace(/\//, "-");
        archiveName = `${prefix}-${platform.replace("/", "_")}.tar.gz`;
      } else {
        archiveName = `${github.context.repo.repo}-${platform.replace("/", "_")}.tar.gz`;
      }

      core.info(`Creating archive ${archiveName}`);
      await archive(archiveName, path);
      assets.push(archiveName);
    }

    const releaseName = gitTag;
    const octokit = github.getOctokit(token);

    core.info(`Creating release ${releaseName}`);
    const release = await octokit.rest.repos.createRelease({
      owner: github.context.repo.owner,
      repo: github.context.repo.repo,
      tag_name: gitTag,
      name: releaseName,
      body: "",
      draft: false,
      prerelease: false,
    });

    for (const asset of assets) {
      core.info(`Uploading asset ${asset}`);
      await octokit.rest.repos.uploadReleaseAsset({
        owner: github.context.repo.owner,
        repo: github.context.repo.repo,
        release_id: release.data.id,
        name: asset,
        mediaType: {
          format: "application/gzip",
        },
        data: fs.readFileSync(asset),
      });
    }
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};

/**
 * Get the blueprint for a project
 * @param {string} project  The name of the project to get the blueprint for
 * @returns {object}        The blueprint object
 */
async function getBlueprint(project) {
  const result = await exec.getExecOutput("forge", ["dump", project]);
  return JSON.parse(result.stdout);
}

/**
 * Generates tags for the given project
 * @param {string} project  The name of the project to get tags for
 * @returns {object}        The tags object
 */
async function getTags(project) {
  const result = await exec.getExecOutput("forge", [
    "-vv",
    "tag",
    "--ci",
    project,
  ]);
  return JSON.parse(result.stdout);
}

/**
 * Archive a directory
 * @param {string} name  The name of the archive
 * @param {string} path  The path to archive
 */
async function archive(name, path) {
  await exec.exec("tar", ["-C", path, "-czf", name, "."]);
}
