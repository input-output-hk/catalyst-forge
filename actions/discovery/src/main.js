const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const absolute = core.getBooleanInput("absolute", { required: false });
    const path = core.getInput("path", { required: true });
    const filters = core.getInput("filters", { required: false });

    await runDeploymentScan(absolute, path);
    await runEarthfileScan(filters, absolute, path);
    await runReleaseScan(absolute, path);
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};

/**
 * Runs the deployment scan
 * @param {boolean} absolute Whether to use absolute paths or not
 * @param {string} path The path to scan
 */
async function runDeploymentScan(absolute, path) {
  const args = ["-vv", "scan", "blueprint", "--filter", "project.deployment"];

  if (absolute === true) {
    args.push("--absolute");
  }
  args.push(path);

  core.info(`Running forge ${args.join(" ")}`);
  const result = await exec.getExecOutput("forge", args);
  const json = JSON.parse(result.stdout);

  core.info(`Found deployments: ${Object.keys(json)}`);
  core.setOutput("deployments", JSON.stringify(Object.keys(json)));
}

/**
 * Runs the earthfile scan
 * @param {string} filters The filters input string
 * @param {boolean} absolute Whether to use absolute paths or not
 * @param {string} path The path to scan
 */
async function runEarthfileScan(filters, absolute, path) {
  let args = ["-vv", "scan", "earthfile", "--enumerate"];

  if (absolute === true) {
    args.push("--absolute");
  }

  args = args.concat(filtersToArgs(filters));
  args.push(path);

  core.info(`Running forge ${args.join(" ")}`);
  const result = await exec.getExecOutput("forge", args);

  core.info(`Found earthfiles: ${result.stdout}`);
  core.setOutput("earthfiles", result.stdout);
}

/**
 * Runs the release scan
 * @param {boolean} absolute Whether to use absolute paths or not
 * @param {string} path The path to scan
 */
async function runReleaseScan(absolute, path) {
  const args = ["-vv", "scan", "blueprint", "--filter", "project.release"];

  if (absolute === true) {
    args.push("--absolute");
  }
  args.push(path);

  core.info(`Running forge ${args.join(" ")}`);
  const result = await exec.getExecOutput("forge", args);
  const json = JSON.parse(result.stdout);

  const releaseMap = Object.entries(json).flatMap(([project, value]) =>
    Object.keys(value["project.release"]).map((name) => ({ project, name })),
  );

  core.info(`Found releases: ${JSON.stringify(releaseMap)}`);
  core.setOutput("releases", JSON.stringify(releaseMap));
}

/**
 * Converts the filters input string to command line arguments.
 * @param {string} input The filters input string
 * @returns {string[]} The filters as command line arguments
 */
function filtersToArgs(input) {
  const lines = input.trim().split("\n");

  const result = [];
  for (const line of lines) {
    result.push("--filter", line);
  }

  return result;
}
