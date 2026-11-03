<?php
namespace SvnAdmin\Controllers;

class PermissionController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isAccessPathViewActive() || (!$appEngine->isUserViewActive() && !$appEngine->isGroupViewActive())) {
            $appEngine->forwardInvalidModule(true);
        }
        $appEngine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN);
        $appTR->loadModule("permissionassign");
        
        // Form request.
        if (check_request_var('assign')) {
            $appEngine->handleAction('assign_usergrouptoaccesspath');
        }
        
        // Basic view data.
        $users = array();
        $groups = array();
        $paths = array();
        
        if ($appEngine->isUserViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_USER, ACL_ACTION_VIEW)) {
            $users = $appEngine->getUserViewProvider()->getUsers();
            usort($users, array('\SvnAdmin\Entities\User', "compare"));
        }
        
        if ($appEngine->isGroupViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_VIEW)) {
            $groups = $appEngine->getGroupViewProvider()->getGroups();
            usort($groups, array('\SvnAdmin\Entities\Group', "compare"));
        }
        
        $paths = $appEngine->getAccessPathViewProvider()->getPaths();
        
        // Filter access-paths for project-managers.
        if ($appEngine->isAuthenticationActive()) {
            $currentUsername = $appEngine->getSessionUsername();
            if ($appEngine->getAclManager()->isUserAccessPathManager($currentUsername)) {
                $paths = $appEngine->getAclManager()->filterAccessPathsList($currentUsername, $paths);
            }
        }
        
        usort($paths, array('\SvnAdmin\Entities\AccessPath', "compare"));
        
        $this->setValue("PermNone", \SvnAdmin\Entities\Permission::$PERM_NONE);
        $this->setValue("PermRead", \SvnAdmin\Entities\Permission::$PERM_READ);
        $this->setValue("PermReadWrite", \SvnAdmin\Entities\Permission::$PERM_READWRITE);
        $this->setValue("UserList", $users);
        $this->setValue("GroupList", $groups);
        $this->setValue("AccessPathList", $paths);
        $this->processTemplate("permission/permissionassign.html.php");
        
        return $this->response;
    }
}
