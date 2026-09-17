import { defineConfig } from "oxlint";

export default defineConfig({
	ignorePatterns: [
		".agents/**",
		".claude/**",
		"tools/oxlint/anti-slop/**",
		"**/dist/**",
		"**/node_modules/**",
	],
	jsPlugins: [
		{ name: "anti-slop", specifier: "./tools/oxlint/anti-slop/index.ts" },
	],
	rules: {
		"anti-slop/no-chained-type-assertions": "error",
		"anti-slop/no-widen-then-assert": "error",
		"anti-slop/no-unsafe-dictionary-type": "error",
		"anti-slop/require-safety-comment-for-type-assertion": "error",
		"anti-slop/no-known-value-widening": "error",
	},
});
