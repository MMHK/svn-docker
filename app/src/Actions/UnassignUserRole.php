<?php
namespace SvnAdmin\Actions;

class UnassignUserRole extends Action
{
    public function execute(): void
    {
        // Check module.
        $this->engine->forwardInvalidModule(!$this->engine->isAclManagerActive());

        // Selected users on page.
        $selusers = get_request_var("selected_users");

        // Selected roles on page.
        $selroles = get_request_var("selected_roles");

        // Validate the selection.
        if ($selusers == NULL || $selroles == NULL)
        {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one user and one role.")));
        }
        else
        {
        	try {
        	  // Count of: All, Done, Failed
        	  $cntAll = count($selroles) * count($selusers);

        	  // Iterate all selected users and roles.
        	  for ($i=0; $i<count($selroles); $i++)
        	  {
        	    $oR = new \SvnAdmin\Entities\Role;
        	    $oR->name = $selroles[$i];

        	    for ($j=0; $j<count($selusers); $j++)
        	    {
        	      $oU = new \SvnAdmin\Entities\User;
        	      $oU->name = $selusers[$j];

        	      if ($this->engine->getAclManager()->removeUserFromRole($oU, $oR))
        	      {
        	      	$this->engine->addMessage(tr("The user %0 has been removed from role %1", array($oU->name, $oR->name)));
        	      }
        	      else
        	      {
        	      	$this->engine->addException(tr("Can not remove user %0 from role %1", array($oU->name, $oR->name)));
        	      }
        	    } //for
        	  } //for

        	  if (!$this->engine->getAclManager()->save())
        	    ;//throw new Exception("Could not save ACL.");
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
