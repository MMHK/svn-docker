<?php
namespace SvnAdmin\Controllers;

class ErrorController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $this->checkAuthentication();
        $appTR->loadModule("errors");
        
        $e = get_request_var('e');
        
        $page = "";
        switch ($e) {
            case ERROR_INVALID_MODULE:
                $page = "error/invalid-module.html.php";
                break;
                
            case ERROR_NO_ACCESS:
                $acl_module = get_request_var("m");
                $acl_action = get_request_var("a");
                $page = "error/no-access.html.php";
                $this->setValue("Module", $acl_module);
                $this->setValue("Action", $acl_action);
                break;
                
            default:
                $appEngine->forward(PAGE_HOME, true);
                break;
        }
        
        if (!empty($page)) {
            $this->processTemplate($page);
        }
        
        return $this->response;
    }
}
