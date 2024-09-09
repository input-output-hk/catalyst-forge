const core = require("@actions/core");
const exec = require("@actions/exec");
const github = require("@actions/github");

async function run() {
  try {
    const project = core.getInput("project", { required: true });
    const path = core.getInput("path", { required: true });
    const platform = core.getInput("platform", { required: false });
    const token = core.getInput("github_token", { required: false });

    const blueprint = await getBlueprint(project);

    const gitTag = parseGitTag(process.env.GITHUB_REF);
    if (gitTag !== "") {
      core.info(`Detected Git tag: ${gitTag}`);
      const projectCleaned = project.replace(/^\.\//, "").replace(/\/$/, "");
      const tag = parseGitMonorepoTag(
        gitTag,
        projectCleaned,
        blueprint.global?.ci?.tagging?.aliases,
      );

      if (tag === "") {
        core.info(`Skipping tag as it does not match the project`);
        return;
      }
    } else {
      core.setFailed("No Git tag detected");
      return;
    }

    let archiveName = "";
    if (gitTag.split("/").length > 1) {
      const prefix = gitTag
        .split("/")
        .slice(0, -1)
        .join("/")
        .replace(/\//, "-");
      archiveName = `${prefix}-${platform}.tar.gz`;
    } else {
      archiveName = `${github.context.repo.repo}-${platform}.tar.gz`;
    }

    core.info(`Creating archive ${archiveName}`);
    await archive(archiveName, path);

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

    core.info(`Uploading asset ${archiveName}`);
    await octokit.rest.repos.uploadReleaseAsset({
      owner: github.context.repo.owner,
      repo: github.context.repo.repo,
      release_id: release.data.id,
      name: archiveName,
      mediaType: {
        format: "application/gzip",
      },
      data: require("fs").readFileSync(archiveName),
    });
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
 * Parse a Git tag from a ref. If the ref is not a tag, an empty string is returned.
 * @param {string} ref The ref to parse
 * @returns {string}   The tag or an empty string
 */
function parseGitTag(ref) {
  if (ref.startsWith("refs/tags/")) {
    return ref.slice(10);
  } else {
    return "";
  }
}

/**
 * Parse a Git mono-repo tag
 * @param {*} tag      The tag to parse
 * @param {*} project  The project path
 * @param {*} aliases  The tag aliases to use
 * @returns {string}   The parsed tag or an empty string
 */
function parseGitMonorepoTag(tag, project, aliases) {
  const parts = tag.split("/");
  if (parts.length > 1) {
    const path = parts.slice(0, -1).join("/");
    const monoTag = parts[parts.length - 1];

    core.info(
      `Detected mono-repo tag path=${path} tag=${monoTag} currentProject=${project}`,
    );
    if (aliases && Object.keys(aliases).length > 0) {
      if (aliases[path] === project) {
        return monoTag;
      }
    }

    if (path === project) {
      return monoTag;
    } else {
      return "";
    }
  } else {
    core.info("Detected non mono-repo tag. Using tag as is.");
    return tag;
  }
}

/**
 * Archive a directory
 * @param {string} name  The name of the archive
 * @param {string} path  The path to archive
 */
async function archive(name, path) {
  await exec.exec("tar", ["-C", path, "-czf", name, "."]);
}
