<?php
namespace SvnAdmin\Controllers;

class UserController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        // Default to list action.
        return $this->list();
    }
    
    public function list(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isUserViewActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_VIEW);
        $appTR->loadModule("userlist");
        $appTR->loadModule("roles");
        
        // Form request to delete users.
        if (check_request_var("delete")) {
            $appEngine->handleAction("delete_user");
        } elseif (check_request_var("assign_role")) {
            $appEngine->handleAction("assign_userrole");
        }
        
        // Assign roles.
        $allroles = null;
        if ($appEngine->isAclManagerActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_ROLE, ACL_ACTION_ASSIGN)) {
            $allroles = $appEngine->getAclManager()->getRoles();
            usort($allroles, array('\SvnAdmin\Entities\Role', "compare"));
        }
        
        // Get all users.
        $users = $appEngine->getUserViewProvider()->getUsers();
        usort($users, array('\SvnAdmin\Entities\User', "compare"));
        
        $this->setValue("UserList", $users);
        $this->setValue("RoleList", $allroles);
        $this->processTemplate("user/userlist.html.php");
        
        return $this->response;
    }
    
    public function create(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isUserEditActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_ADD);
        $appTR->loadModule("usercreate");
        
        // Form request to create user.
        if (check_request_var('create')) {
            $appEngine->handleAction('create_user');
        }
        
        $this->processTemplate("user/usercreate.html.php");
        
        return $this->response;
    }
    
    public function view(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_VIEW);
        if (!$appEngine->isGroupViewActive() && !$appEngine->isAccessPathViewActive()) {
            $appEngine->forwardInvalidModule(true);
        }
        $appTR->loadModule("userview");
        $appTR->loadModule("roles");
        
        // Action handling.
        if (check_request_var('unassign')) {
            $appEngine->handleAction('unassign_usergroup');
        } elseif (check_request_var('unassign_permission')) {
            $appEngine->handleAction('unassign_permission');
        } elseif (check_request_var('assign_role')) {
            $appEngine->handleAction('assign_userrole');
        } elseif (check_request_var('unassign_role')) {
            $appEngine->handleAction('unassign_userrole');
        } elseif (check_request_var('unassign_projectmanager')) {
            $appEngine->handleAction('unassign_projectmanager');
        } elseif (check_request_var('assign_usergroup')) {
            $appEngine->handleAction('assign_usertogroup');
        }
        
        // Get required variables.
        $usernameEnc = get_request_var('username');
        $username = rawurldecode($usernameEnc);
        
        $oUser = new \SvnAdmin\Entities\User();
        $oUser->id = $username;
        $oUser->name = $username;
        
        // Roles.
        $allroles = null;
        $rolesOfUser = null;
        if ($appEngine->isAclManagerActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_ROLE, ACL_ACTION_VIEW)) {
            $rolesOfUser = $appEngine->getAclManager()->getRolesOfUser($oUser);
            
            if ($appEngine->checkUserAuthentication(false, ACL_MOD_ROLE, ACL_ACTION_ASSIGN)) {
                $allroles = $appEngine->getAclManager()->getRoles();
                usort($allroles, array('\SvnAdmin\Entities\Role', "compare"));
                
                $rolesOfUserLen = count($rolesOfUser);
                for ($i = 0; $i < $rolesOfUserLen; $i++) {
                    if_array_remove_object_element($allroles, $rolesOfUser[$i], "name");
                }
                $allroles = array_values($allroles);
            }
        }
        
        // Groups.
        $groups = null;
        $allgroups = null;
        if ($appEngine->isGroupViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_VIEW)) {
            $groups = $appEngine->getGroupViewProvider()->getGroupsOfUser($oUser);
            usort($groups, array('\SvnAdmin\Entities\Group', "compare"));
            
            if ($appEngine->isGroupViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_ASSIGN)) {
                $allgroups = $appEngine->getGroupViewProvider()->getGroups();
                usort($allgroups, array('\SvnAdmin\Entities\Group', "compare"));
                $len = count($groups);
                for ($i = 0; $i < $len; $i++) {
                    if_array_remove_object_element($allgroups, $groups[$i], "name");
                }
                $allgroups = array_values($allgroups);
            }
        }
        
        // Access-Path permissions.
        $paths = null;
        if ($appEngine->isAccessPathViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW)) {
            $paths = $appEngine->getAccessPathViewProvider()->getPathsOfUser($oUser);
            usort($paths, array('\SvnAdmin\Entities\AccessPath', "compare"));
        }
        
        // Project manager check.
        $restricted_paths = null;
        $isprojectmanager = false;
        if ($appEngine->isAuthenticationActive()) {
            $uname = $username;
            if ($appEngine->getAclManager()->isUserAccessPathManager($uname)) {
                $restricted_paths = $appEngine->getAclManager()->getAccessPathsOfUser($uname);
                if (count($restricted_paths) > 0) {
                    usort($restricted_paths, array('\SvnAdmin\Entities\AccessPath', "compare"));
                    $isprojectmanager = true;
                }
            }
        }
        
        $this->setValue("RoleList", $rolesOfUser);
        $this->setValue("RoleListAll", $allroles);
        $this->setValue("GroupList", $groups);
        $this->setValue("GroupListAll", $allgroups);
        $this->setValue("PathList", $paths);
        $this->setValue("RestrictedPathList", $restricted_paths);
        $this->setValue("ProjectManager", $isprojectmanager);
        $this->setValue("Username", $username);
        $this->setValue("UsernameEncoded", rawurlencode($username));
        $this->processTemplate("user/userview.html.php");
        
        return $this->response;
    }
    
    public function changePassword(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $appEngine->forwardInvalidModule(!$appEngine->isUserEditActive());
        $appEngine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_CHANGEPASS);
        $appTR->loadModule("userchangepass");
        
        // Action handling.
        if (check_request_var('changepass')) {
            $appEngine->handleAction('change_password');
        }
        
        $encUsername = get_request_var("username");
        $username = rawurldecode($encUsername);
        
        $sessUser = $appEngine->getSessionUsername();
        if ($sessUser != null) {
            if ($username == null) {
                $username = $sessUser;
            }
            
            if ($sessUser != $username) {
                $appEngine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_CHANGEPASS_OTHER);
            }
        }
        
        $this->setValue("Username", $username);
        $this->processTemplate("user/userchangepassword.html.php");
        
        return $this->response;
    }
    
    public function groupAssign(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isUserViewActive() || !$appEngine->isGroupViewActive() || !$appEngine->isGroupEditActive()) {
            $appEngine->forwardInvalidModule(true);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_GROUP, ACL_ACTION_ASSIGN);
        $appTR->loadModule("usergroupassign");
        
        // Form request.
        if (check_request_var('assign')) {
            $appEngine->handleAction('assign_usertogroup');
        }
        
        // User list.
        $users = null;
        if ($appEngine->isUserViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_USER, ACL_ACTION_VIEW)) {
            $users = $appEngine->getUserViewProvider()->getUsers();
            
            $o = new \SvnAdmin\Entities\User();
            $o->id = '*';
            $o->name = '*';
            
            $users = remove_item_by_value($users, $o, true);
            usort($users, array('\SvnAdmin\Entities\User', "compare"));
        }
        
        // Group list.
        $groups = null;
        if ($appEngine->isGroupViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_VIEW)) {
            $groups = $appEngine->getGroupViewProvider()->getGroups();
            usort($groups, array('\SvnAdmin\Entities\Group', "compare"));
        }
        
        $this->setValue("UserList", $users);
        $this->setValue("GroupList", $groups);
        $this->processTemplate("group/membership.html.php");
        
        return $this->response;
    }
}
