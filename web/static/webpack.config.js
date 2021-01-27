/**
 * Copyright 2020 The Logsuck Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const CopyPlugin = require("copy-webpack-plugin");

module.exports = {
  devtool: "source-map",
  plugins: [
    new CopyPlugin({
      patterns: [
        { from: "./template.html", to: "./dist/template.html" },
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
    filename: "dist/[name].js",
  },
  entry: {
    home: "./src/pages/home.tsx",
    search: "./src/pages/search.tsx",
  },
  resolve: {
    extensions: [".ts", ".tsx", ".js"],
  },
};
