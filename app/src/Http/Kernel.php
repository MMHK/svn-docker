<?php
namespace SvnAdmin\Http;

class Kernel
{
    private Router $router;
    
    public function __construct(Router $router)
    {
        $this->router = $router;
    }
    
    public function handle(Request $request): Response
    {
        try {
            return $this->router->dispatch($request);
        } catch (\Exception $e) {
            $response = new Response();
            $response->setStatusCode(500);
            $response->setBody('500 Internal Server Error: ' . $e->getMessage());
            return $response;
        }
    }
    
    public function terminate(Response $response): void
    {
        $response->send();
    }
}
