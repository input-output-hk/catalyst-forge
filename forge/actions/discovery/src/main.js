const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const absolute = core.getBooleanInput("absolute", { required: false });
    const enumerate = core.getBooleanInput("enumerate", { required: false });
    const path = core.getInput("path", { required: true });
    const filters = core.getInput("filters", { required: false });

    let args = ["-vv", "scan"];

    if (absolute === true) {
      args.push("--absolute");
    }

    if (enumerate === true) {
      args.push("--enumerate");
    }

    args = args.concat(filtersToArgs(filters));
    args.push(path);

    core.info(`Running forge ${args.join(" ")}`);

    let stdout = "";
    const options = {};
    options.listeners = {
      stdout: (data) => {
        stdout += data.toString();
      },
    };

    await exec.exec("forge", args, options);

    core.setOutput("result", stdout);
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};

function filtersToArgs(input) {
  const lines = input.trim().split("\n");

  const result = [];
  for (const line of lines) {
    result.push("--filter", line);
  }

  return result;
}
