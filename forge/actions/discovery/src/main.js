const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const path = core.getInput("path", { required: true });
    const filters = core.getInput("filters", { required: false });

    const args = ["-vv", "scan"];
    args.concat(filtersToArgs(filters));
    args.push(path);

    core.info(`Running forge ${args.join(" ")}`);

    let stdout = "";
    const options = {};
    options.listeners = {
      stdout: (data) => {
        stdout += data.toString();
      },
      stderr: (data) => {
        console.log(data.toString());
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
    result.push("-f", line);
  }

  return result;
}
