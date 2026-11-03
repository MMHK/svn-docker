<?php
namespace SvnAdmin\Actions;

class UnassignPermission extends Action
{
    public function execute(): void
    {
        //
        // Authentication
        //

        if (!$this->engine->isProviderActive(PROVIDER_ACCESSPATH_EDIT)) {
        	$this->engine->forwardError(ERROR_INVALID_MODULE);
        }

        $this->engine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN);

        //
        // HTTP Request Vars
        //

        $selusers = get_request_var('selected_users');
        $selgroups = get_request_var('selected_groups');
        $selpaths = get_request_var('selected_accesspaths');

        //
        // Validation
        //

        if ($selpaths == NULL
        	|| ($selgroups == NULL && $selusers == NULL)) {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one access-path and one user or group.")));
        }
        else {
        	try {
        		$selpathsLen = count($selpaths);
        		$selusersLen = ($selusers != null) ? count($selusers) : 0;
        		$selgroupsLen = ($selgroups != null) ? count($selgroups) : 0;

        		// iterate all selected_accesspaths
        		for ($i = 0; $i < $selpathsLen; ++$i) {
        			$oAP = new \SvnAdmin\Entities\AccessPath($selpaths[$i]);

        			// Is the user restricted to some paths? (project-manager)
        			if ($this->engine->isAuthenticationActive()) {
        				$currentUsername = $this->engine->getSessionUsername();
        				if ($this->engine->getAclManager()->isUserAccessPathManager($currentUsername)) {
        					if (!$this->engine->getAclManager()->isUserAdminOfPath($currentUsername, $selpaths[$i])) {
        						$this->engine->addException(new Exception(tr("You don't have the permission to unassign the access path %0", array($oAP->path))));
        						continue;
        					}
        				}
        			}

        			// iterate selected_users.
        			for ($iu = 0; $iu < $selusersLen; ++$iu) {
        				if (empty($selusers[$iu])) {
        					continue;
        				}

        				$oU = new \SvnAdmin\Entities\User($selusers[$iu], $selusers[$iu]);

        				// remove user from ap
        				try {
        					if ($this->engine->getAccessPathEditProvider()->removeUserFromAccessPath($oU, $oAP))
        						$this->engine->addMessage(tr("Removed user %0 from access path %1", array($oU->name, $oAP->path)));
        					else
        						$this->engine->addException(new Exception(tr("Can not remove user %0 from access path %1", array($oU->name, $oAP->path))));
        				}
        				catch (Exception $e) {
        					$this->engine->addException($e);
        				}
        			}

        			// iterate selected_groups.
        			for ($ig = 0; $ig < $selgroupsLen; ++$ig) {
        				if (empty($selgroups[$ig])) {
        					continue;
        				}

        				$oG = new \SvnAdmin\Entities\Group($selgroups[$ig], $selgroups[$ig]);

        				// remove group from ap
        				try {
        					if ($this->engine->getAccessPathEditProvider()->removeGroupFromAccessPath($oG, $oAP))
        						$this->engine->addMessage(tr("Removed group %0 from access path %1", array($oG->name, $oAP->path)));
        					else
        						$this->engine->addException(new Exception(tr("Can not remove group %0 from access path %1", array($oG->name, $oAP->path))));
        				}
        				catch (Exception $e) {
        					$this->engine->addException($e);
        				}
        			}
        		}
        		$this->engine->getAccessPathEditProvider()->save();
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
