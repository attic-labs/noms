'use strict';

const webpack = require('webpack');

const devMode = process.env.NODE_ENV !== 'production';

// Replaces all process.env.foo with the value of the enviroment variable.
function replaceEnv(envVars) {
  const replacements = {
    'process.env.NODE_ENV': JSON.stringify(String(process.env.NODE_ENV)),
  };
  for (const name of envVars) {
    if (!(name in process.env)) {
      console.error(`Missing required environment variable: ${name}`);  //eslint-disable-line
      process.exit(-1);
    }
    replacements[`process.env.${name}`] = JSON.stringify(process.env[name]);
  }
  return new webpack.DefinePlugin(replacements);
}

function exitStatus() {
  this.plugin('done', function(stats) {
    if (stats.compilation.errors && stats.compilation.errors.length) {
        console.error(stats.compilation.errors);
        process.exit(1);
    }
  });
}

function getPlugins(envVars) {
  const plugins = [replaceEnv(envVars)];
  if (!devMode) {
    plugins.push(exitStatus);
    plugins.push(new webpack.optimize.UglifyJsPlugin({
      compress: {
        warnings: false,
        screw_ie8: true,  //eslint-disable-line
      },
    }));
  }
  return plugins;

}

// Anything that uses |options| in |module.exports| must be a function or getter.
let options = {};

function shouldInclude(p) {
  // Files in the src directory of the project.
  return /\/src\//.test(p) && !/node_modules/.test(p);
}

module.exports = {
  get module() {
    return {
      loaders: [{
        test: /\.js$/,
        loader: 'babel-loader',
        include: shouldInclude,
      }],
    };
  },
  get plugins() {
    return getPlugins(options.requiredEnvVars || []);
  },
  devtool: devMode ? '#inline-source-map' : '',
  watch: devMode,
  configure(newOptions) {
    options = newOptions;
    return this;
  },
};
