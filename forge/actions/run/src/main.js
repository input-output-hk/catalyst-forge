const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const artifact = core.getBooleanInput("artifact", { required: false });
    const local = core.getBooleanInput("local", { required: false });
    const path = core.getInput("path", { required: true });

    let args = ["-vv", "run"];

    if (artifact !== "") {
      args.push("--artifact", artifact);
    }

    if (local === true) {
      args.push("--local");
    }

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
