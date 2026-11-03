<?php
namespace SvnAdmin\Http;

class Router
{
    private array $routes = [];
    private array $legacyRoutes = [];
    
    public function get(string $path, callable $handler): void
    {
        $this->routes['GET'][$path] = $handler;
    }
    
    public function post(string $path, callable $handler): void
    {
        $this->routes['POST'][$path] = $handler;
    }
    
    public function any(string $path, callable $handler): void
    {
        $this->routes['GET'][$path] = $handler;
        $this->routes['POST'][$path] = $handler;
    }
    
    public function legacy(string $oldPath, string $newPath): void
    {
        $this->legacyRoutes[$oldPath] = $newPath;
    }
    
    public function dispatch(Request $request): Response
    {
        $uri = $request->getUri();
        $method = $request->getMethod();
        
        // Parse URI (remove query string)
        $path = parse_url($uri, PHP_URL_PATH);

        // 應用掛在 /svnadmin（SCRIPT_NAME 的目錄）之下，路由表則以相對於應用根的
        // 路徑註冊（/users、/index.php…），這裡先剝掉掛載前綴再比對
        $basePath = str_replace('\\', '/', dirname($_SERVER['SCRIPT_NAME'] ?? ''));
        $basePath = rtrim($basePath, '/');
        if ($basePath !== '' && ($path === $basePath || str_starts_with($path, $basePath . '/'))) {
            $path = substr($path, strlen($basePath));
            if ($path === '') {
                $path = '/';
            }
        }
        if ($path === '') {
            $path = '/';
        }
        
        // Check legacy routes first (301 redirect)
        if (isset($this->legacyRoutes[$path])) {
            $response = new Response();
            $newPath = $this->legacyRoutes[$path];
            // Preserve query string
            $queryString = parse_url($uri, PHP_URL_QUERY);
            if ($queryString) {
                $newPath .= '?' . $queryString;
            }
            return $response->redirect($newPath, 301);
        }
        
        // Match route
        if (isset($this->routes[$method][$path])) {
            $handler = $this->routes[$method][$path];
            return $handler($request);
        }
        
        // 404
        $response = new Response();
        $response->setStatusCode(404);
        $response->setBody('404 Not Found');
        return $response;
    }
}
