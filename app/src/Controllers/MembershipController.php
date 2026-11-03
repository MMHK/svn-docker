<?php
namespace SvnAdmin\Controllers;

class MembershipController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
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
