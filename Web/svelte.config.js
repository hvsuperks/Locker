import adapter from '@sveltejs/adapter-static'; // QUAN TRỌNG: Đổi cái này
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// adapter-auto chỉ dùng cho các host như Vercel/Netlify
		// adapter-static sẽ tạo ra các file tĩnh cho Go nhúng vào
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html', // Hoặc index.html nếu bạn làm SPA
			precompress: false,
			strict: true
		})
	}
};

export default config;