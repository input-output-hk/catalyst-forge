const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const path = core.getInput("path", { required: true });
    const filters = core.getInput("filters", { required: false });

    const args = ["-vv", "scan"];
    args.push(filtersToArgs(filters));
    args.push(path);

    core.info(`Running forge ${args.join(" ")}`);

    const options = {};
    const stdout = "";
    options.listeners = {
      stdout: (data) => {
        stdout += data.toString();
      },
    };

    await exec.exec('forge', args, options);

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
  lines.forEach((line) => {
    result.push("-f", line);
  });

  return result;
}
