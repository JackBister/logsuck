const CopyPlugin = require("copy-webpack-plugin");

module.exports = {
  devtool: "source-map",
  plugins: [
    new CopyPlugin({
      patterns: [
        { from: "./index.html", to: "./dist/index.html" },
        { from: "./style.css", to: "./dist/style.css" },
        {
          from: "./node_modules/bootstrap/dist/css/bootstrap.min.css",
          to: "./dist/bootstrap.min.css",
        },
        {
          from: "./node_modules/bootstrap/dist/js/bootstrap.min.js",
          to: "./dist/bootstrap.min.js",
        },
        {
          from: "./node_modules/popper.js/dist/umd/popper.min.js",
          to: "./dist/popper.min.js",
        },
        {
          from: "./node_modules/jquery/dist/jquery.min.js",
          to: "./dist/jquery.min.js",
        },
      ],
    }),
  ],
  module: {
    rules: [{ test: /\.tsx?$/, loader: "ts-loader" }],
  },
  output: {
    path: __dirname,
    filename: "dist/app.js",
  },
  resolve: {
    extensions: [".ts", ".tsx", ".js"],
  },
};
