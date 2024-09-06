const core = require("@actions/core");
const forge = require("../../lib/src/forge");

async function run() {
  try {
    const artifact = core.getInput("artifact", { required: false });
    const local = core.getBooleanInput("local", { required: false });
    const path = core.getInput("path", { required: true });

    const args = ["-vv", "run"];

    if (artifact !== "") {
      args.push("--artifact", artifact);
    }

    if (local === true) {
      args.push("--local");
    }

    args.push(path);

    core.info(`Running forge ${args.join(" ")}`);
    const result = await forge.runForge(args);

    core.setOutput("result", result.stdout);
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};
