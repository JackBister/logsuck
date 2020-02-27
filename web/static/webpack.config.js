module.exports = {
    devtool: 'source-map',
    module: {
        rules: [
            { test: /\.tsx?$/, loader: 'ts-loader' }
        ]
    },
    output: {
        path: __dirname,
        filename: 'app.js'
    },
    resolve: {
        extensions: ['.ts', '.tsx', '.js']
    }
};
