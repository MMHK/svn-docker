<?php
namespace SvnAdmin\Controllers;

class GroupController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        return $this->list();
    }
    
    public function list(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isGroupViewActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_GROUP, ACL_ACTION_VIEW);
        $appTR->loadModule("grouplist");
        
        // Form 'delete' request.
        if (check_request_var('delete')) {
            $appEngine->handleAction('delete_group');
        }
        
        // Get all groups.
        $groups = $appEngine->getGroupViewProvider()->getGroups();
        usort($groups, array('\SvnAdmin\Entities\Group', "compare"));
        
        $this->setValue("GroupList", $groups);
        $this->processTemplate("group/grouplist.html.php");
        
        return $this->response;
    }
    
    public function create(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isGroupEditActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_GROUP, ACL_ACTION_ADD);
        $appTR->loadModule("groupcreate");
        
        // Form request to create group.
        if (check_request_var('create')) {
            $appEngine->handleAction('create_group');
        }
        
        $this->processTemplate("group/groupcreate.html.php");
        
        return $this->response;
    }
    
    public function view(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isGroupViewActive() && !$appEngine->isAccessPathViewActive()) {
            $appEngine->forwardInvalidModule(true);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_GROUP, ACL_ACTION_VIEW);
        $appTR->loadModule("groupview");
        
        // Action handling.
        if (check_request_var('unassign')) {
            $appEngine->handleAction('unassign_usergroup');
        }
        if (check_request_var('unassign_permission')) {
            $appEngine->handleAction('unassign_permission');
        }
        if (check_request_var('assign_usergroup')) {
            $appEngine->handleAction("assign_usertogroup");
        }
        
        // Get required variables.
        $groupname = get_request_var('groupname');
        
        $oGroup = new \SvnAdmin\Entities\Group();
        $oGroup->id = $groupname;
        $oGroup->name = $groupname;
        
        // Users of group.
        $users = null;
        $allusers = null;
        if ($appEngine->isGroupViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_VIEW)) {
            $users = $appEngine->getGroupViewProvider()->getUsersOfGroup($oGroup);
            usort($users, array('\SvnAdmin\Entities\User', "compare"));
            
            if ($appEngine->isUserViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_ASSIGN)) {
                $allusers = $appEngine->getUserViewProvider()->getUsers(false);
                usort($allusers, array('\SvnAdmin\Entities\User', "compare"));
                $len = count($users);
                for ($i = 0; $i < $len; $i++) {
                    if_array_remove_object_element($allusers, $users[$i], "name");
                }
                $allusers = array_values($allusers);
            }
        }
        
        // Access-Path permissions.
        $paths = null;
        if ($appEngine->isAccessPathViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW)) {
            $paths = $appEngine->getAccessPathViewProvider()->getPathsOfGroup($oGroup);
            usort($paths, array('\SvnAdmin\Entities\AccessPath', "compare"));
        }
        
        $this->setValue("GroupName", $groupname);
        $this->setValue("GroupNameEncoded", rawurlencode($groupname));
        $this->setValue("UserList", $users);
        $this->setValue("AllUserList", $allusers);
        $this->setValue("AccessPathList", $paths);
        $this->processTemplate("group/groupview.html.php");
        
        return $this->response;
    }
}
