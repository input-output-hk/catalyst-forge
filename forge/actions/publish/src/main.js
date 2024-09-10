const core = require("@actions/core");
const exec = require("@actions/exec");
const github = require("@actions/github");

async function run() {
  try {
    const project = core.getInput("project", { required: true });
    const image = core.getInput("image", { required: true });

    const exists = await imageExists(image);
    if (!exists) {
      core.error(
        `Unable to find image '${image}' in the local Docker daemon. Did you add a 'container' and 'tag' argument to your target?`,
        {
          file: `${project}/Earthfile`,
        },
      );
      core.setFailed(
        `Image '${image}' does not exist in the local Docker daemon`,
      );
      return;
    }

    const currentBranch = github.context.ref.replace("refs/heads/", "");
    const defaultBranch = github.context.payload.repository.default_branch;
    core.info(
      `Current ref: ${currentBranch}\nDefault branch: ${defaultBranch}`,
    );
    if (currentBranch !== defaultBranch) {
      core.info("Not on default branch, skipping publish");
      return;
    }

    const blueprint = await getBlueprint(project);

    if (blueprint?.project?.container === undefined) {
      core.warning(
        `Project '${project}' does not have a container defined. Skipping publish`,
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
    const result = await getTags(project);
    if (result.git !== "") {
      tags.push(result.git);
    }
    tags.push(result.generated);

    const container = blueprint.project.container;
    const registries = blueprint.global.ci.registries;

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
    "--trim",
    project,
  ]);
  return JSON.parse(result.stdout);
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

/***
 * Push a Docker image to a registry
 * @param {string} image  The name of the image to push
 * @returns {Promise<int>} The exit code of the command
 */
async function pushImage(image) {
  await exec.exec("docker", ["push", image]);
}

/**
 * Tag a Docker image
 * @param {string} oldImage  The old image name
 * @param {string} newImage  The new image name
 */
async function tagImage(oldImage, newImage) {
  await exec.exec("docker", ["tag", oldImage, newImage]);
}
