#!/usr/bin/env node
import { createGzip } from 'node:zlib';
import { pipeline, Readable, Writable } from 'node:stream';
import { promisify } from 'node:util';
import { readdir, stat, writeFile, mkdir, readFile } from 'node:fs/promises';
import { join, extname } from 'node:path';

const pipe = promisify(pipeline);

async function gzipSize(filePath) {
	const data = await readFile(filePath);
	const gz = createGzip({ level: 6 });
	const chunks = [];
	await pipe(Readable.from(data), gz, new Collector(chunks));
	return Buffer.concat(chunks).length;
}

class Collector extends Writable {
	constructor(arr) {
		super();
		this.arr = arr;
	}
	_write(chunk, _enc, cb) {
		this.arr.push(chunk);
		cb();
	}
}

async function walk(dir) {
	const out = [];
	const entries = await readdir(dir, { withFileTypes: true });
	for (const e of entries) {
		const p = join(dir, e.name);
		if (e.isDirectory()) out.push(...(await walk(p)));
		else out.push(p);
	}
	return out;
}

async function main() {
	const clientDir = '.svelte-kit/output/client';
	try {
		await stat(clientDir);
	} catch {
		console.error('Client output not found. Run `vite build` first.');
		process.exit(1);
	}

	const files = await walk(clientDir);
	const jsFiles = files.filter((f) => extname(f) === '.js');
	const cssFiles = files.filter((f) => extname(f) === '.css');

	let jsBytes = 0;
	let cssBytes = 0;
	for (const f of jsFiles) jsBytes += await gzipSize(f);
	for (const f of cssFiles) cssBytes += await gzipSize(f);

	const result = {
		generatedAt: new Date().toISOString(),
		clientDir,
		totals: {
			jsGzipBytes: jsBytes,
			cssGzipBytes: cssBytes,
			jsGzipKB: +(jsBytes / 1024).toFixed(2),
			cssGzipKB: +(cssBytes / 1024).toFixed(2)
		},
		budgets: {
			jsGzipKB: 80,
			cssGzipKB: 10
		},
		withinBudgets: jsBytes / 1024 <= 80 && cssBytes / 1024 <= 10
	};

	await mkdir('dist', { recursive: true });
	await writeFile('dist/analysis.json', JSON.stringify(result, null, 2));

	console.log('Analysis written to dist/analysis.json');
	console.log(
		'Totals (gzip): JS ~',
		result.totals.jsGzipKB,
		'KB; CSS ~',
		result.totals.cssGzipKB,
		'KB'
	);
	console.log('Within budgets:', result.withinBudgets);
}

main().catch((e) => {
	console.error(e);
	process.exit(1);
});
