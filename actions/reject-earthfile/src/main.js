const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const filters = core.getInput("filters", { required: true });
    const rootPath = core.getInput("root-path", { required: false }) || ".";
    const filterSource =
      core.getInput("filter-source", { required: false }) || "targets";
    const verbosity = core.getInput("verbosity", { required: false }) || "info";

    const filterPairs = filters
      .split("\n")
      .filter((line) => line.trim() !== "");
    const expressions = [];
    const messages = [];

    for (const pair of filterPairs) {
      const [expr, ...msgParts] = pair.split(",");
      if (expr && msgParts.length > 0) {
        expressions.push(expr.trim());
        messages.push(msgParts.join(",").trim());
      }
    }

    if (expressions.length === 0) {
      core.setFailed("No valid filter expressions provided");
      return;
    }

    core.info(`Found ${expressions.length} filter expressions to check`);

    let hasRejections = false;
    let rejectionOutput = "";

    for (let i = 0; i < expressions.length; i++) {
      const expr = expressions[i];
      const msg = messages[i];

      let verbosityLevel;
      switch (verbosity) {
        case "error":
          verbosityLevel = "-v";
          break;
        case "info":
          verbosityLevel = "-vv";
          break;
        case "debug":
          verbosityLevel = "-vvv";
          break;
      }

      core.info(`Checking filter: ${expr}`);
      const forgeArgs = [
        "scan",
        verbosityLevel,
        "earthfile",
        "--filter",
        expr,
        "--filter-source",
        filterSource,
        "--enumerate",
        "--combine",
        "--pretty",
        rootPath,
      ];

      core.info(`Running: forge ${forgeArgs.join(" ")}`);

      try {
        const result = await exec.getExecOutput("forge", forgeArgs, {
          silent: true,
        });

        let jsonResult;
        try {
          jsonResult = JSON.parse(result.stdout);
        } catch (parseError) {
          core.warning(
            `Failed to parse forge output for filter "${expr}": ${parseError.message}`,
          );
          jsonResult = [];
        }

        if (Array.isArray(jsonResult) && jsonResult.length > 0) {
          hasRejections = true;

          const sortedPaths = jsonResult.sort();

          rejectionOutput += `❌ ${msg}:\n`;
          for (const path of sortedPaths) {
            rejectionOutput += `- ${path}\n`;
          }
          rejectionOutput += "\n";
        }
      } catch (execError) {
        core.warning(
          `Failed to execute forge command for filter "${expr}": ${execError.message}`,
        );
      }
    }

    // Fail if any rejections were found
    if (hasRejections) {
      core.setFailed(rejectionOutput);
    } else {
      core.info("No Earthfiles matched the filter expressions");
    }
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};
