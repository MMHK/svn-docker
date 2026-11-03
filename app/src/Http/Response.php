<?php
namespace SvnAdmin\Http;

class Response
{
    private int $statusCode = 200;
    private array $headers = [];
    private string $body = '';
    
    public function setStatusCode(int $code): self
    {
        $this->statusCode = $code;
        return $this;
    }
    
    public function setHeader(string $name, string $value): self
    {
        $this->headers[$name] = $value;
        return $this;
    }
    
    public function setBody(string $body): self
    {
        $this->body = $body;
        return $this;
    }
    
    public function redirect(string $url, int $code = 302): self
    {
        $this->statusCode = $code;
        $this->headers['Location'] = $url;
        return $this;
    }
    
    public function send(): void
    {
        http_response_code($this->statusCode);
        foreach ($this->headers as $name => $value) {
            header("$name: $value");
        }
        echo $this->body;
    }
}
