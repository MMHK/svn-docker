<?php
namespace SvnAdmin\Http;

class Request
{
    private string $method;
    private string $uri;
    private array $query;
    private array $post;
    private array $cookies;
    private array $server;
    
    public function __construct()
    {
        $this->method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
        $this->uri = $_SERVER['REQUEST_URI'] ?? '/';
        $this->query = $_GET;
        $this->post = $_POST;
        $this->cookies = $_COOKIE;
        $this->server = $_SERVER;
    }
    
    public function getMethod(): string
    {
        return $this->method;
    }
    
    public function getUri(): string
    {
        return $this->uri;
    }
    
    public function getQuery(string $key, $default = null)
    {
        return $this->query[$key] ?? $default;
    }
    
    public function getPost(string $key, $default = null)
    {
        return $this->post[$key] ?? $default;
    }
    
    public function getCookie(string $key, $default = null)
    {
        return $this->cookies[$key] ?? $default;
    }
    
    public function getServer(string $key, $default = null)
    {
        return $this->server[$key] ?? $default;
    }
    
    public function getAllQuery(): array
    {
        return $this->query;
    }
    
    public function getAllPost(): array
    {
        return $this->post;
    }
    
    public function isPost(): bool
    {
        return $this->method === 'POST';
    }
    
    public function isGet(): bool
    {
        return $this->method === 'GET';
    }
}
