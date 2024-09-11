const core = require("@actions/core");
const exec = require("@actions/exec");
const github = require("@actions/github");

async function run() {
  try {
    const project = core.getInput("project");
    const image = core.getInput("image");
    const skip_branch_check = core.getBooleanInput("skip_branch_check");
    const target = core.getInput("target");

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
    }

    const images = {};
    if (platforms.length > 0) {
      for (const platform of platforms) {
        images[platform] = `${image}_${platform.replace("/", "_")}`;
      }
    } else {
      images["default"] = image;
    }

    for (const image of Object.values(images)) {
      core.info(`Validating image ${image} exists`);
      const exists = await imageExists(image);
      if (!exists) {
        core.error(
          `Unable to find image '${image}' in the local Docker daemon. Did you add a 'container' and 'tag' argument to your target?`,
          {
            file: `${project}/Earthfile`,
          },
        );
        core.setFailed(`Unable to find image: ${image}`);
        return;
      }
    }

    const currentBranch = github.context.ref.replace("refs/heads/", "");
    const defaultBranch = github.context.payload.repository.default_branch;
    if (
      currentBranch !== defaultBranch &&
      !skip_branch_check &&
      !github.context.ref.includes("refs/tags/")
    ) {
      core.info("Not on default branch, skipping publish");
      return;
    }

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
        `The repository does not have any container registries defined. Skipping publish`,
      );
      return;
    }

    const tags = [];
    const result = await getTags(project);
    if (result.git !== "") {
      tags.push(result.git);
    } else {
      tags.push(result.generated);
    }

    const container = blueprint.project.container;
    const registries = blueprint.global.ci.registries;

    if (platforms.length > 0) {
      for (const registry of registries) {
        for (const tag of tags) {
          const pushed = [];
          for (const platform of platforms) {
            const existingImage = images[platform];
            const taggedImage = `${registry}/${container}:${tag}_${platform.replace("/", "_")}`;

            core.info(`Tagging image ${existingImage} as ${taggedImage}`);
            await tagImage(existingImage, taggedImage);

            core.info(`Pushing image ${taggedImage}`);
            await pushImage(taggedImage);

            pushed.push(taggedImage);
          }

          const multiImage = `${registry}/${container}:${tag}`;
          core.info(`Creating multi-platform image ${multiImage}`);
          await exec.exec("docker", [
            "buildx",
            "imagetools",
            "create",
            "--tag",
            multiImage,
            ...pushed,
          ]);
        }
      }
    } else {
      const image = images["default"];
      for (const registry of registries) {
        for (const tag of tags) {
          const taggedImage = `${registry}/${container}:${tag}`;

          core.info(`Tagging image ${image} as ${taggedImage}`);
          await tagImage(image, taggedImage);

          core.info(`Pushing image ${taggedImage}`);
          await pushImage(taggedImage);
        }
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
