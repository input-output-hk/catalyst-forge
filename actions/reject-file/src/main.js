const core = require("@actions/core");
const exec = require("@actions/exec");

async function run() {
  try {
    const filters = core.getInput("filters", { required: true });
    const maxPaths = core.getInput("max-paths", { required: false }) || "15";
    const rootPath =
      core.getInput("root-path", { required: false }) ||
      process.env.GITHUB_WORKSPACE ||
      ".";
    const verbosity = core.getInput("verbosity", { required: false }) || "info";

    // Split rules by double newline
    const rules = filters.split("\n\n").filter((rule) => rule.trim() !== "");
    const filterConfigs = [];

    // Parse each rule
    for (let ruleIndex = 0; ruleIndex < rules.length; ruleIndex++) {
      const lines = rules[ruleIndex]
        .split("\n")
        .filter((line) => line.trim() !== "");

      const filenamePatterns = [];
      const contentPatterns = [];
      let description = null;
      let hasMultipleDescriptions = false;

      // Parse lines in the rule
      for (const line of lines) {
        const trimmedLine = line.trim();

        if (trimmedLine.startsWith("file:")) {
          const pattern = trimmedLine.substring(5).trim();
          if (pattern) {
            filenamePatterns.push(pattern);
          }
        } else if (trimmedLine.startsWith("content:")) {
          const pattern = trimmedLine.substring(8).trim();
          if (pattern) {
            contentPatterns.push(pattern);
          }
        } else if (trimmedLine.startsWith("description:")) {
          if (description !== null) {
            hasMultipleDescriptions = true;
          }
          description = trimmedLine.substring(12).trim();
        } else {
          core.warning(
            `Invalid line in rule ${ruleIndex + 1}: "${trimmedLine}". Lines must start with "file:", "content:", or "description:"`,
          );
        }
      }

      // Validate rule
      if (hasMultipleDescriptions) {
        core.warning(
          `Rule ${ruleIndex + 1} has multiple description lines, skipping`,
        );
        continue;
      }

      if (!description) {
        core.warning(`Rule ${ruleIndex + 1} has no description, skipping`);
        continue;
      }

      if (filenamePatterns.length === 0 && contentPatterns.length === 0) {
        core.warning(
          `Rule ${ruleIndex + 1} has no patterns specified, skipping`,
        );
        continue;
      }

      filterConfigs.push({
        filenamePatterns,
        contentPatterns,
        message: description,
      });
    }

    if (filterConfigs.length === 0) {
      core.setFailed("No valid filter configurations provided");
      return;
    }

    core.info(`Found ${filterConfigs.length} filter configurations to check`);

    let hasRejections = false;
    let rejectionOutput =
      "Some files failed to pass the filter expressions:\n\n";

    for (const config of filterConfigs) {
      const { filenamePatterns, contentPatterns, message } = config;

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

      // Build filter description for logging
      const filterDesc = [];
      if (filenamePatterns.length > 0)
        filterDesc.push(`file patterns: ${filenamePatterns.join(", ")}`);
      if (contentPatterns.length > 0)
        filterDesc.push(`content patterns: ${contentPatterns.join(", ")}`);
      core.info(`Checking filter: ${filterDesc.join("; ")}`);

      const forgeArgs = ["scan", verbosityLevel, "all"];

      // Add all filename patterns
      for (const pattern of filenamePatterns) {
        forgeArgs.push("-f", pattern);
      }

      // Add all content patterns
      for (const pattern of contentPatterns) {
        forgeArgs.push("-c", pattern);
      }

      forgeArgs.push("--pretty", ".");

      core.info(`Running: forge ${forgeArgs.join(" ")} (from ${rootPath})`);

      try {
        const result = await exec.getExecOutput("forge", forgeArgs, {
          silent: true,
          cwd: rootPath,
        });

        let jsonResult;
        try {
          jsonResult = JSON.parse(result.stdout);
        } catch (parseError) {
          core.warning(`Failed to parse forge output: ${parseError.message}`);
          jsonResult = [];
        }

        if (Array.isArray(jsonResult) && jsonResult.length > 0) {
          hasRejections = true;

          const sortedPaths = jsonResult.sort();
          const maxPathsToShow = parseInt(maxPaths); // Limit output to prevent truncation

          rejectionOutput += `❌ ${message}:\n`;
          for (
            let i = 0;
            i < Math.min(sortedPaths.length, maxPathsToShow);
            i++
          ) {
            // Clean up path by removing leading ./
            let cleanPath = sortedPaths[i];
            if (cleanPath.startsWith("./")) {
              cleanPath = cleanPath.substring(2);
            }
            rejectionOutput += `  - ${cleanPath}\n`;
          }

          if (sortedPaths.length > maxPathsToShow) {
            rejectionOutput += `  ... and ${sortedPaths.length - maxPathsToShow} more files\n`;
          }

          rejectionOutput += "\n";
        }
      } catch (execError) {
        core.warning(`Failed to execute forge command: ${execError.message}`);
      }
    }

    if (hasRejections) {
      core.setFailed(rejectionOutput);
    } else {
      core.info("No files matched the filter expressions");
    }
  } catch (error) {
    core.setFailed(error.message);
  }
}

module.exports = {
  run,
};
