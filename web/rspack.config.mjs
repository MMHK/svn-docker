import path from 'node:path';
import { fileURLToPath } from 'node:url';
import rspack from '@rspack/core';
import { VueLoaderPlugin } from 'rspack-vue-loader';

const rootDir = path.dirname(fileURLToPath(import.meta.url));

// The Go binary embeds api/pkg/webui/dist (pkg/webui/embed.go), so the build
// writes straight into the Go tree. dist is committed only as build output.
const distDir = path.resolve(rootDir, '../api/pkg/webui/dist');

/**
 * rspack configuration (plan section 4.4).
 *
 * - history-mode SPA mounted at /svnadmin/;
 * - production builds have no source map and therefore no `eval`, so the CSP
 *   from plan section 5.7 (`script-src 'self'`) holds;
 * - Tailwind v4 is compiled through postcss-loader + @tailwindcss/postcss on
 *   top of Rspack's native CSS support.
 */
export default (env = {}, argv = {}) => {
  const mode = argv.mode ?? (process.env.NODE_ENV === 'development' ? 'development' : 'production');
  const isDev = mode === 'development';

  return {
    mode,
    entry: './src/main.ts',
    output: {
      path: distDir,
      publicPath: '/svnadmin/',
      filename: 'assets/[name].[contenthash:8].js',
      chunkFilename: 'assets/[name].[contenthash:8].js',
      assetModuleFilename: 'assets/[name].[contenthash:8][ext]',
      clean: true,
    },
    // No eval in any build: production must be CSP-safe.
    devtool: isDev ? 'source-map' : false,
    resolve: {
      extensions: ['.ts', '.tsx', '.js', '.mjs', '.vue', '.json'],
      alias: {
        '@': path.resolve(rootDir, 'src'),
      },
    },
    module: {
      rules: [
        {
          test: /\.vue$/,
          loader: 'rspack-vue-loader',
          options: {
            experimentalInlineMatchResource: true,
          },
        },
        {
          test: /\.ts$/,
          exclude: /node_modules/,
          loader: 'builtin:swc-loader',
          options: {
            jsc: {
              parser: { syntax: 'typescript' },
              target: 'es2022',
            },
          },
        },
        {
          test: /\.css$/,
          use: ['postcss-loader'],
          type: 'css',
        },
      ],
    },
    plugins: [
      new VueLoaderPlugin(),
      new rspack.HtmlRspackPlugin({
        template: './index.html',
        title: 'SVN Admin',
      }),
    ],
    devServer: {
      port: 5173,
      hot: true,
      historyApiFallback: true,
      client: {
        overlay: true,
        logging: 'warn',
      },
      proxy: [
        {
          context: ['/svnadmin/api'],
          target: 'http://127.0.0.1:8080',
          changeOrigin: false,
        },
      ],
    },
    stats: 'errors-warnings',
  };
};
