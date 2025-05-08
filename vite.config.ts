import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import { fileURLToPath } from 'url';
import fs from 'fs';
import cssInjectedByJsPlugin from 'vite-plugin-css-injected-by-js';
import { dirname, resolve, join } from 'path';
import { exec } from 'child_process';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

function watchAllShFiles(pluginContext, dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });

  entries.forEach((entry) => {
    const fullPath = join(dir, entry.name);

    if (entry.isDirectory()) {
      watchAllShFiles(pluginContext, fullPath);
    } else if (entry.isFile() && fullPath.endsWith('.sh')) {
      pluginContext.addWatchFile(fullPath);
    }
  });
}

function inlineShellImports(scriptPath, visited = new Set(), isRoot = true) {
  if (visited.has(scriptPath)) {
    return '';
  }
  visited.add(scriptPath);

  let content = fs.readFileSync(scriptPath, 'utf8');
  const dirOfScript = dirname(scriptPath);

  const lines = content.split('\n');
  let output = '';

  for (const line of lines) {
    const match = line.match(/^import\s+(.+)$/);
    if (match) {
      const importedFile = match[1].trim();
      const importAbsolutePath = resolve(dirOfScript, importedFile);
      output += inlineShellImports(importAbsolutePath, visited, false);
    } else {
      if (!isRoot && line.match(/^#/)) {
        continue;
      }
      output += line + '\n';
    }
  }

  return output;
}

export default defineConfig(({ mode }) => {
  const __dirname = dirname(fileURLToPath(import.meta.url));

  const isProduction = mode === 'production';
  const watching = process.env.VITE_WATCH === '1';
  console.log(`Building for ${isProduction ? 'production' : 'development'}...`);

  return {
    root: 'frontend',
    base: './',
    plugins: [
      react(),
      cssInjectedByJsPlugin(),
      {
        buildStart() {
          this.addWatchFile(resolve(__dirname, 'frontend', 'index.html'));
          this.addWatchFile(resolve(__dirname, 'frontend', 'index.asp'));

          const backendDir = resolve(__dirname, 'backend', 'scripts');
          watchAllShFiles(this, backendDir);
        },
        generateBundle(_, bundle) {
          for (const file of Object.values(bundle)) {
            if (file.type === 'chunk') {
              file.code = file.code.replace(/<%/g, '<\\u0025');
            }
          }
        },
        name: 'copy-and-sync',
        closeBundle: () => {
          console.log('Vite finished building. Copying extra files...');

          try {
            const scriptPath = join(__dirname, 'backend', 'scripts', 'idefix.sh');
            const mergedContent = inlineShellImports(scriptPath);

            const distScript = join(__dirname, 'dist', 'idefix.sh');
            fs.writeFileSync(distScript, mergedContent, { mode: 0o755 });

            fs.copyFileSync('frontend/index.asp', 'dist/index.asp');
            console.log('Files copied successfully.');
          } catch (e) {
            console.error('File copy error:', e);
          }

          console.log('Running sync.js script...');
          exec('node vite.sync.js', (error, stdout, stderr) => {
            if (error) {
              console.error('Error running sync.js script:', error);
              return;
            }
            console.log(`Sync script output: ${stdout}`);
            if (stderr) {
              console.error(`Sync script errors: ${stderr}`);
            }
          });
        }
      }
    ],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'frontend')
      }
    },

    build: {
      outDir: './../dist',
      assetsInlineLimit: 200_000,
      rollupOptions: {
        input: resolve(__dirname, 'frontend/index.html'),
        output: {
          entryFileNames: 'app.js'
        }
      },
      emptyOutDir: !watching,
      sourcemap: false,
      minify: true, //watching ? false : "esbuild",
      watch: watching ? {} : null
    }
  };
});
