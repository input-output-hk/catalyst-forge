const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const args = core.getInput("args", { required: false });
    const command = core.getInput("command", { required: true });
    const local = core.getBooleanInput("local", { required: false });
    const targetArgs = core.getInput("target_args", { required: false });
    const verbosity = core.getInput("verbosity", { required: false });

    let verbosityLevel = 0;
    switch (verbosity) {
      case "error":
        verbosityLevel = 1;
        break;
      case "info":
        verbosityLevel = 2;
        break;
      case "debug":
        verbosityLevel = 3;
        break;
    }

    const forgeArgs = ["--ci"];

    if (verbosityLevel > 0) {
      forgeArgs.push(`-${"v".repeat(verbosityLevel)}`);
    }

    if (local === true) {
      forgeArgs.push("--local");
    }

    forgeArgs.push(command);

    if (args !== "") {
      forgeArgs.push(...args.split(" "));
    }

    if (targetArgs !== "") {
      forgeArgs.push("--", ...targetArgs.split(" "));
    }

    core.info(`Running forge ${forgeArgs.join(" ")}`);
    const result = await runForge(forgeArgs);

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
