<?php
namespace SvnAdmin\Actions;

class AssignUserGroupToAccessPath extends Action
{
    public function execute(): void
    {
        $this->engine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN);

        // Required variables.
        $selusers  = get_request_var('selected_users');
        $selgroups = get_request_var('selected_groups');
        $selpaths  = get_request_var('selected_accesspaths');
        $selperm   = get_request_var('permission'); // TODO: There is no check, whether this var is given!

        if (count($selusers) == 1 && empty($selusers[0]))
          $selusers = NULL;
        if (count($selgroups) == 1 && empty($selgroups[0]))
          $selgroups = NULL;
        if (count($selpaths) == 1 && empty($selpaths[0]))
          $selpaths = NULL;

        if( $selpaths == NULL || ( $selusers == NULL && $selgroups == NULL ) )
        {
        	$this->engine->addException(new ValidationException(tr("You have to select a user or group and an access-path to perform this action.")));
        }
        else
        {
        	try {
        	  // The number of selected elements.
        	  $selpathsLen = count($selpaths);
        	  $selgroupsLen = ($selgroups != NULL) ? count($selgroups) : 0;
        	  $selusersLen =  ($selusers  != NULL) ? count($selusers)  : 0;

        	  // Create permission object.
        	  $oP = new \SvnAdmin\Entities\Permission;
        	  $oP->perm = $selperm;

        	  // Iterate all selected_accesspaths.
        	  for( $i=0; $i<$selpathsLen; $i++ )
        	  {
        	    $oAP = new \SvnAdmin\Entities\AccessPath;
        	    $oAP->id = $selpaths[$i];
        	    $oAP->path = $selpaths[$i];

        	    // Is the user restricted to some paths? (project-manager)
        	    if ($this->engine->isAuthenticationActive())
        	    {
        	      $currentUsername = $this->engine->getSessionUsername();
        	      if ($this->engine->getAclManager()->isUserAccessPathManager($currentUsername))
        	      {
        	        if (!$this->engine->getAclManager()->isUserAdminOfPath($currentUsername, $oAP->path))
        	        {
        	          // Skip assignment.
        	          $this->engine->addException(new Exception(tr("No administration permission for %0", array($oAP->path))));
        	          continue;
        	        }
        	      }
        	    }

        		// Iterate selected_users.
        		for( $iu=0; $iu<$selusersLen; $iu++ )
            	{
        			$oU = new \SvnAdmin\Entities\User;
        			$oU->id = $selusers[$iu];
        			$oU->name = $selusers[$iu];

        			try {
        				$b = $this->engine->getAccessPathEditProvider()->assignUserToAccessPath($oU, $oAP, $oP);
        				if (!$b) {
        					throw new Exception("ERROR");
        				}
        				$this->engine->addMessage(tr("Grant %0 permission to %1 on %2", array($oP->perm, $oU->name, $oAP->path)));
        			}
        			catch (Exception $e) {
        				$this->engine->addException($e);
        			}
            	}

        	    // Iterate selected_groups.
        	    for( $ig=0; $ig<$selgroupsLen; $ig++ )
        	    {
        	      $oG = new \SvnAdmin\Entities\Group;
        	      $oG->id = $selgroups[$ig];
        	      $oG->name = $selgroups[$ig];

        	      try {
        	      	$b = $this->engine->getAccessPathEditProvider()->assignGroupToAccessPath( $oG, $oAP, $oP );
        	      	if (!$b) {
        	      		throw new Exception("ERROR");
        	      	}
        	      	$this->engine->addMessage(tr("Grant %0 permission to %1 on %2", array($oP->perm, $oG->name, $oAP->path)));
        	      }
        	      catch (Exception $e) {
        	      	$this->engine->addException($e);
        	      }
        	    }
        	  }

        	  // Save changes!
        	  $this->engine->getAccessPathEditProvider()->save();
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
