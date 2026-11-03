<?php
namespace SvnAdmin\Controllers;

class AccessPathController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        return $this->list();
    }
    
    public function list(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isProviderActive(PROVIDER_ACCESSPATH_VIEW)) {
            $appEngine->forwardError(ERROR_INVALID_MODULE);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW);
        $appTR->loadModule("accesspathslist");
        
        // Actions.
        if (check_request_var("delete")) {
            $appEngine->handleAction("delete_accesspath");
        } elseif (check_request_var("assign_projectmanager")) {
            $appEngine->handleAction("assign_projectmanager");
        }
        
        // View Data.
        $list = array();
        $users = array();
        
        try {
            $list = $appEngine->getAccessPathViewProvider()->getPaths();
            usort($list, array('\SvnAdmin\Entities\AccessPath', "compare"));
            
            if ($appEngine->isAuthenticationActive()) {
                $currentUsername = $appEngine->getSessionUsername();
                if ($appEngine->getAclManager()->isUserAccessPathManager($currentUsername)) {
                    $list = $appEngine->getAclManager()->filterAccessPathsList($currentUsername, $list);
                }
                
                if ($appEngine->isUserViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_PROJECTMANAGER, ACL_ACTION_ASSIGN)) {
                    $users = $appEngine->getUserViewProvider()->getUsers(false);
                    usort($users, array('\SvnAdmin\Entities\User', "compare"));
                }
                
                $listCount = count($list);
                for ($i = 0; $i < $listCount; $i++) {
                    $managers = $appEngine->getAclManager()->getUsersOfAccessPath($list[$i]->path);
                    $list[$i]->managers = $managers;
                }
            }
        } catch (\Exception $ex) {
            $appEngine->addException($ex);
        }
        
        $this->setValue("UserList", $users);
        $this->setValue("AccessPathList", $list);
        $this->processTemplate("accesspath/accesspathlist.html.php");
        
        return $this->response;
    }
    
    public function create(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isAccessPathEditActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_ADD);
        $appTR->loadModule("accesspathcreate");
        
        // Form request to create.
        if (check_request_var('create')) {
            $appEngine->handleAction('create_accesspath');
        }
        
        // HTTP request parameters.
        $varRepoEnc = get_request_var("r");
        $varPathEnc = get_request_var("p");
        $varRepo = rawurldecode($varRepoEnc);
        $varPath = rawurldecode($varPathEnc);
        
        $preFill = "";
        if ($varRepo != null) {
            $preFill = $varRepo . ":/";
            if ($varPath != null) {
                $preFill .= $varPath;
            }
        }
        
        $this->setValue("DefaultAccessPath", $preFill);
        $this->processTemplate("accesspath/accesspathcreate.html.php");
        
        return $this->response;
    }
    
    public function view(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isAccessPathViewActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW);
        $appTR->loadModule("accesspathview");
        
        // Action handling.
        if (check_request_var("unassign")) {
            $appEngine->handleAction("unassign_permission");
        } elseif (check_request_var("assign_permission")) {
            $appEngine->handleAction("assign_usergrouptoaccesspath");
        }
        
        // Get required variables.
        $accesspathEnc = get_request_var('accesspath');
        $accesspath = rawurldecode($accesspathEnc);
        
        $o = new \SvnAdmin\Entities\AccessPath();
        $o->path = $accesspath;
        
        $users = $appEngine->getAccessPathViewProvider()->getUsersOfPath($o);
        $groups = $appEngine->getAccessPathViewProvider()->getGroupsOfPath($o);
        
        // Data to assign new permissions.
        if ($appEngine->isAccessPathEditActive() && $appEngine->checkUserAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN)) {
            if ($appEngine->isUserViewActive()) {
                $allusers = $appEngine->getUserViewProvider()->getUsers();
                usort($allusers, array('\SvnAdmin\Entities\User', "compare"));
                $this->setValue("UserListAll", $allusers);
            }
            
            if ($appEngine->isGroupViewActive()) {
                $allgroups = $appEngine->getGroupViewProvider()->getGroups();
                usort($allgroups, array('\SvnAdmin\Entities\Group', "compare"));
                $this->setValue("GroupListAll", $allgroups);
            }
        }
        
        $this->setValue("PermNone", \SvnAdmin\Entities\Permission::$PERM_NONE);
        $this->setValue("PermRead", \SvnAdmin\Entities\Permission::$PERM_READ);
        $this->setValue("PermReadWrite", \SvnAdmin\Entities\Permission::$PERM_READWRITE);
        $this->setValue("UserList", $users);
        $this->setValue("GroupList", $groups);
        $this->setValue("AccessPath", $accesspath);
        $this->setValue("AccessPathEncoded", rawurlencode($accesspath));
        $this->processTemplate("accesspath/accesspathview.html.php");
        
        return $this->response;
    }
}
