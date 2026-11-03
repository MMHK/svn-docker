<?php
namespace SvnAdmin\Controllers;

class RoleController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isAclManagerActive()) {
            $appEngine->forwardInvalidModule(true);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_ROLE, ACL_ACTION_VIEW);
        $appTR->loadModule("roles");
        
        $roles = $appEngine->getAclManager()->getRoles();
        
        $this->setValue('RoleList', $roles);
        $this->processTemplate('role/rolelist.html.php');
        
        return $this->response;
    }
}
