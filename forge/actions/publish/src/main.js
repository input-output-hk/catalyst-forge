const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const project = core.getInput("project", { required: true });
    const image = core.getInput("image", { required: true });

    if (!imageExists(image)) {
      core.setFailed(`Image ${image} does not exist in the local Docker daemon`);
      return;
    }

    const blueprint = await getBlueprint(project);

    if (blueprint?.project?.container === undefined) {
      core.warning(`Project ${project} does not have a container defined. Skipping publish`);
      return;
    } else if (blueprint?.global?.tagging?.strategy === undefined) {
      core.warning(`The repository does not have a tagging strategy defined. Skipping publish`);
      return;
    } else if (blueprint?.global?.registry === undefined || blueprint?.global?.registry.length === 0) {
      core.warning(`The repository does not have any registries defined. Skipping publish`);
      return;
    }

    const container = blueprint.project.container;
    const registries = blueprint.global.registry;
    const tag = getTag(blueprint.global.tagging.strategy);

    for (const registry of registries) {
      const taggedImage = `${registry}/${container}:${tag}`;

      core.info(`Tagging image ${image} as ${taggedImage}`);
      await tagImage(image, taggedImage);

      core.info(`Pushing image ${taggedImage}`);
      await pushImage(taggedImage);
    }
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};

/**
 *
 * @param {string} project  The name of the project to get the blueprint for
 * @returns {object}        The blueprint object
 */
async function getBlueprint(project) {
  return JSON.parse(await exec.getExecOutput("forge", ["blueprint", "dump", project]));
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
  ret = await exec.exec("docker", ["inspect", name]);
  return ret === 0;
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