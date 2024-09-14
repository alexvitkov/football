const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const { DefinePlugin } = require('webpack');
const { readFileSync } = require('fs');
const { exit } = require('process');

let config;
try {
	config = readFileSync('../../../config.json')
	config = JSON.parse(config);
} catch {
	console.error("Failed to read config.json");
	exit(1);
}

module.exports = {
	entry: './main.ts',
	mode: 'development',
	devtool: 'source-map',
	target: ['web', 'es7'],
	module: {
		rules: [
			{
				test: /\.tsx?$/,
				use: 'ts-loader',
				//exclude: /node_modules/,
			},
		],
	},
	resolve: {
		extensions: ['.tsx', '.ts', '.js'],
	},
	output: {
		filename: 'bundle.js',
		path: path.resolve(__dirname, '../../../build/front/games/football'),
		library: ['lib'],
	},
	devServer: {
		static: {
			directory: path.join(__dirname, './assets'),
		},
		compress: true,
		port: 8080,
		allowedHosts: 'all',
	},
	plugins: [
		new HtmlWebpackPlugin({
			template: path.join(__dirname, 'index.html'),
			title: "DAS KAPITAL",
		}),
		new DefinePlugin({
			GAME_WS: `"${config.wsSchema}://${config.host}${(config.lobbyPort != 80 && config.lobbyPort != 443) ? `:${config.lobbyPort}` : ""}/api/gameWs"`,
		})
],
};