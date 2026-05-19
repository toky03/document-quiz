import express from 'express';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const server = express();
const serverDistFolder = dirname(fileURLToPath(import.meta.url));
const browserDistFolder = resolve(serverDistFolder, '../browser');
const indexHtml = join(serverDistFolder, 'index.server.html');

server.set('view engine', 'html');
server.set('views', browserDistFolder);

server.get(/.*\..*/, express.static(browserDistFolder, { maxAge: '1y' }));

server.get(/.*/, (req, res) => {
  res.render(indexHtml, { req });
});

const port = Number.parseInt(process.env['PORT'] ?? '4000', 10);
server.listen(port, () => {
  console.log(`Node Express server listening on http://localhost:${port}`);
});
