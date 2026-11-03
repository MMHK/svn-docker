<?php
namespace SvnAdmin\Controllers;

class HomeController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $this->checkAuthentication();
        $appTR->loadModule("index");
        $appTR->loadModule("roles");
        
        $phpVersion = phpversion();
        $roles = null;
        if ($appEngine->isAuthenticationActive()) {
            $u = new \SvnAdmin\Entities\User();
            $u->name = $appEngine->getSessionUsername();
            $roles = $appEngine->getAclManager()->getRolesOfUser($u);
            sort($roles);
        }
        
        $this->setValue("PHPVersion", $phpVersion);
        $this->setValue("Roles", $roles);
        $this->processTemplate("index.html.php");
        
        return $this->response;
    }
}
