const exec = require("@actions/exec");

/**
 * Run the forge command with the given arguments
 * @param {Array<string>} args Arguments to pass to the forge command
 * @returns {Promise<exec.ExecOutput>} The output of the forge command
 */
async function runForge(args) {
    return await exec.getExecOutput("forge", args);
}

module.exports = {
    runForge,
};