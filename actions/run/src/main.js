const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const artifact = core.getInput("artifact", { required: false });
    const local = core.getBooleanInput("local", { required: false });
    const path = core.getInput("path", { required: true });
    const targetArgs = core.getInput("target_args", { required: false });

    const args = ["-vv", "run", "--ci"];

    if (artifact !== "") {
      args.push("--artifact", artifact);
    }

    if (local === true) {
      args.push("--local");
    }

    args.push(path);

    if (targetArgs !== "") {
      args.push("--", ...targetArgs.split(" "));
    }

    core.info(`Running forge ${args.join(" ")}`);
    const result = await runForge(args);

    core.setOutput("result", result.stdout);
  } catch (error) {
    core.setFailed(error.message);
  }
}

async function runForge(args) {
  return await exec.getExecOutput("forge", args);
}

module.exports = {
  run,
};
