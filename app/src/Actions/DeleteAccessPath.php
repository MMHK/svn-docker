<?php
namespace SvnAdmin\Actions;

class DeleteAccessPath extends Action
{
    public function execute(): void
    {
        //
        // Authentication
        //

        if (!$this->engine->isAccessPathEditActive())
        {
        	$this->engine->forwardInvalidModule(true);
        }

        $this->engine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_DELETE);

        //
        // HTTP Request Vars
        //

        $selected = get_request_var('selected_accesspaths');

        //
        // Validation
        //

        if ($selected == NULL)
        {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one access-path.")));
        }
        else
        {
        	for ($i = 0; $i < count($selected); $i++)
        	{
        		$ap = new \SvnAdmin\Entities\AccessPath();
        		$ap->path = $selected[$i];

        		// Is the user restricted to some paths? (project-manager)
        		if ($this->engine->isAuthenticationActive())
        		{
        			$currentUsername = $this->engine->getSessionUsername();
        			if ($this->engine->getAclManager()->isUserAccessPathManager($currentUsername))
        			{
        				if (!$this->engine->getAclManager()->isUserAdminOfPath($currentUsername, $ap->getPath()))
        				{
        					$this->engine->addException(new Exception(tr('No permission to handle Access-Path: %0', array($ap->getPath()))));
        					continue;
        				}
        			}
        		}

        		// Remove all project-manager assignments to the access-path.
        		if ($this->engine->isAclManagerActive())
        		{
        			$this->engine->getAclManager()->removeAssignmentsToPath($ap->getPath());
        		}

        		// Remove the access-path.

        		if (!$this->engine->getAccessPathEditProvider()->deleteAccessPath($ap))
        		{
        			$this->engine->addException(new Exception(tr('Can not delete Access-Path: %0', array($ap->getPath()))));
        		}
        		$this->engine->addMessage(tr('Removed Access-Path: %0', array($ap->getPath())));
        	}

        	// Save changes.

        	$b = $this->engine->getAccessPathEditProvider()->save();
        	if (!$b)
        	{
        		$this->engine->addException(new Exception('Can not save changes to Access-Path-Edit-Provider'));
        	}

        	if ($this->engine->isAclManagerActive())
        	{
        		$b = $this->engine->getAclManager()->save();
        		if (!$b)
        		{
        			$this->engine->addException(new Exception('Can not save changes to Acl-Manager'));
        		}
        	}
        }
    }
}
