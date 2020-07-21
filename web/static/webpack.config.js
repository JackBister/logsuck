const CopyPlugin = require('copy-webpack-plugin');

module.exports = {
    devtool: 'source-map',
    plugins: [
        new CopyPlugin({
            patterns: [
                { from: './index.html', to: './dist/index.html' },
                { from: './style.css', to: './dist/style.css' }
            ]
        })
    ],
    module: {
        rules: [
            { test: /\.tsx?$/, loader: 'ts-loader' }
        ]
    },
    output: {
        path: __dirname,
        filename: 'dist/app.js'
    },
    resolve: {
        extensions: ['.ts', '.tsx', '.js']
    }
};
