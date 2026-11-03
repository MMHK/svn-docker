<?php
namespace SvnAdmin\Actions;

class DeleteUser extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule(!$this->engine->isUserEditActive());
        $this->engine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_DELETE);

        $selusers = get_request_var('selected_users');

        if ($selusers == NULL)
        {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one user.")));
        }
        else
        {
        	try {
        	  $doneList = array();
        	  $warnList = array();
        	  $failedList = array();
        	  $selusersCount = count($selusers);

        	  for($i=0; $i<$selusersCount; $i++)
        	  {
        	    $u = new \SvnAdmin\Entities\User;
        	    $u->id = $selusers[$i];
        	    $u->name = $selusers[$i];

        	    // Skip * user.
        	    if ($u->name == "*")
        	    {
        	      continue;
        	    }

        	    // Remove user from groups.
        	    if ($this->engine->isGroupEditActive())
        	    {
        	      $this->engine->getGroupEditProvider()->removeUserFromAllGroups($u);
        	    }

        	    // Remove user from Access-Path's.
        	    if ($this->engine->isAccessPathEditActive())
        	    {
        	      $this->engine->getAccessPathEditProvider()->removeUserFromAllAccessPaths($u);
        	    }

        	    // Remove project-manager associations.
        	    if ($this->engine->isAclManagerActive())
        	    {
        	      if ($this->engine->getAclManager()->isUserAccessPathManager($u->name))
        	      {
        	        if (!$this->engine->getAclManager()->deleteAccessPathAdmin($u->name))
        	        {
        	          $this->engine->addException(new Exception(tr("Could not remove user's Access-Path-Manager associations:"+$u->name)));
        	        }
        	      }
        	    }

        	    // Remove roles of user.
        	    if ($this->engine->isAclManagerActive())
        	    {
        	      if ($this->engine->getAclManager()->removeAllRolesFromUser($u->name))
        	      {
        	      }
        	    }

        	    // Delete the user.
        	    $done = $this->engine->getUserEditProvider()->deleteUser($u);
        	    if ($done)
        	    {
        	    	$this->engine->addMessage(tr("Removed user %0 successfully.", array($u->name)));
        	    }
        	    else
        	    {
        	    	$this->engine->addException(tr("Can not remove user %0."));
        	    }
        	  } // for(users)

        	  $this->engine->getUserEditProvider()->save();

        	  if ($this->engine->isGroupEditActive())
        	    $this->engine->getGroupEditProvider()->save();

        	  if ($this->engine->isAccessPathEditActive())
        	    $this->engine->getAccessPathEditProvider()->save();

        	  if ($this->engine->isAclManagerActive())
        	    $this->engine->getAclManager()->save();

        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
