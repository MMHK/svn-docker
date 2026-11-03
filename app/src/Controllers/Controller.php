<?php
namespace SvnAdmin\Controllers;

use SvnAdmin\Http\Request;
use SvnAdmin\Http\Response;

abstract class Controller
{
    protected Request $request;
    protected Response $response;
    
    public function __construct(Request $request, Response $response)
    {
        $this->request = $request;
        $this->response = $response;
    }
    
    protected function setValue(string $key, $value): void
    {
        global $appTemplate;
        $appTemplate->addReplacement($key, $value);
    }
    
    protected function processTemplate(string $template): void
    {
        global $appTemplate;
        ob_start();
        include __DIR__ . '/../../pages/' . $template;
        $content = ob_get_clean();
        $this->response->setBody($content);
    }
    
    protected function redirect(string $url, int $code = 302): Response
    {
        return $this->response->redirect($url, $code);
    }
    
    protected function checkAuthentication(bool $redirect = true): bool
    {
        global $appEngine;
        return $appEngine->checkUserAuthentication($redirect);
    }
    
    abstract public function handle(): Response;
}
