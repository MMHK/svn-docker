// 把純手寫的 JS 資源複製到 app/public/assets/（階段 5 改由 Dockerfile 建置期產生）
import { copyFileSync, mkdirSync } from 'node:fs';

mkdirSync('app/public/assets', { recursive: true });
copyFileSync('assets/app.js', 'app/public/assets/app.js');
console.log('copied assets/app.js -> app/public/assets/app.js');
