const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const project = core.getInput("project", { required: true });
    const image = core.getInput("image", { required: true });

    const exists = await imageExists(image);
    if (!exists) {
      core.setFailed(
        `Image '${image}' does not exist in the local Docker daemon`,
      );
      return;
    }

    const blueprint = await getBlueprint(project);

    if (blueprint?.project?.container === undefined) {
      core.warning(
        `Project '${project}' does not have a container defined. Skipping publish`,
      );
      return;
    } else if (blueprint?.global?.ci?.tagging?.strategy === undefined) {
      core.warning(
        `The repository does not have a tagging strategy defined. Skipping publish`,
      );
      return;
    } else if (
      blueprint?.global?.ci?.registries === undefined ||
      blueprint?.global?.ci?.registries.length === 0
    ) {
      core.warning(
        `The repository does not have any registries defined. Skipping publish`,
      );
      return;
    }

    const tags = [];
    const gitTag = parseGitTag(process.env.GITHUB_REF);
    if (gitTag !== "") {
      core.info(`Detected Git tag: ${gitTag}`);
      const projectCleaned = project.replace(/^\.\//, "").replace(/\/$/, "");
      const tag = parseGitMonorepoTag(
        gitTag,
        projectCleaned,
        blueprint.global?.ci?.tagging?.aliases,
      );

      if (tag !== "") {
        tags.push(tag);
      }
    } else {
      core.info("No Git tag detected");
    }

    const container = blueprint.project.container;
    const registries = blueprint.global.ci.registries;
    tags.push(getTag(blueprint.global.ci.tagging.strategy));

    for (const registry of registries) {
      for (const tag of tags) {
        const taggedImage = `${registry}/${container}:${tag}`;

        core.info(`Tagging image ${image} as ${taggedImage}`);
        await tagImage(image, taggedImage);

        core.info(`Pushing image ${taggedImage}`);
        await pushImage(taggedImage);
      }
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
  const result = await exec.getExecOutput("forge", [
    "blueprint",
    "dump",
    project,
  ]);
  return JSON.parse(result.stdout);
}

/**
 * Get the tag to use for the Docker image from the tagging strategy
 * @param {string} strategy  The tagging strategy to use
 * @returns {string}         The tag to use
 */
function getTag(strategy) {
  switch (strategy) {
    case "commit": {
      return strategyCommit();
    }
    default: {
      throw new Error(`Unknown tagging strategy: ${strategy}`);
    }
  }
}

/**
 * Check if a Docker image exists
 * @param {string} name  The name of the image to check
 * @return {boolean}     True if the image exists, false otherwise
 */
async function imageExists(name) {
  const result = await exec.exec("docker", ["inspect", name], {
    ignoreReturnCode: true,
    silent: true,
  });

  console.log(`Result: ${result}`);

  return result === 0;
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
      core.info(`Skipping tag as it does not match the project`);
      return "";
    }
  } else {
    core.info("Detected non mono-repo tag. Using tag as is.");
    return tag;
  }
}

/***
 * Push a Docker image to a registry
 */
async function pushImage(image) {
  await exec.exec("docker", ["push", image]);
}

/**
 * The "commit" tagging strategy
 * @returns {string} The commit hash
 */
function strategyCommit() {
  return process.env.GITHUB_SHA;
}

/**
 * Tag a Docker image
 * @param {string} oldImage  The old image name
 * @param {string} newImage  The new image name
 */
async function tagImage(oldImage, newImage) {
  await exec.exec("docker", ["tag", oldImage, newImage]);
}
