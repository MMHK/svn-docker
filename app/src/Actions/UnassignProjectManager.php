<?php
namespace SvnAdmin\Actions;

class UnassignProjectManager extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule(!$this->engine->isAuthenticationActive());
        $this->engine->checkUserAuthentication(true, ACL_MOD_PROJECTMANAGER, ACL_ACTION_UNASSIGN);

        $selusers = get_request_var("selected_users");
        $selpaths = get_request_var("selected_accesspaths");

        if ($selusers == null || $selpaths == null)
        {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one access-path and one user.")));
        }
        else
        {
        	try {
        	  $selusersCount = count($selusers);
        	  $selpathsCount = count($selpaths);

        	  $doneList = array();
        	  $failedList = array();

        	  for ($i=0; $i<$selusersCount; $i++)
        	  {
        	    for ($j=0; $j<$selpathsCount; $j++)
        	    {
        	      $b = $this->engine->getAclManager()->removeAccessPathAdmin($selpaths[$j], $selusers[$i]);
        	      if ($b)
        	      	$this->engine->addMessage(tr("Removed Project-Manager status of user %0 from %1", array($selusers[$i], $selpaths[$j])));
        	      else
        	      	$this->engine->addException(new Exception(tr("Can not remove Project-Manager status of user %0 from %1", array($selusers[$i], $selpaths[$j]))));
        	    }
        	  }
        	  $this->engine->getAclManager()->save();
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
