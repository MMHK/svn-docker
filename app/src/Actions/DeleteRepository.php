<?php
namespace SvnAdmin\Actions;

class DeleteRepository extends Action
{
    public function execute(): void
    {
        //
        // Authentication
        //

        if (!$this->engine->isProviderActive(PROVIDER_REPOSITORY_EDIT)) {
        	$this->engine->forwardError(ERROR_INVALID_MODULE);
        }

        // Disabled by config?
        if (!($this->engine->getConfig()->getValueAsBoolean('GUI', 'RepositoryDeleteEnabled', true))) {
        	$this->engine->forwardError(ERROR_INVALID_MODULE);
        }

        //
        // HTTP Request Vars
        //

        $varParentIdentifierEnc = get_request_var('pi');
        $selrepos = get_request_var("selected_repos");
        $remove_accesspaths = check_request_var('delete_ap');

        $varParentIdentifier = rawurldecode($varParentIdentifierEnc);

        //
        // Validation
        //

        if ($selrepos == NULL || $varParentIdentifier === NULL) {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one repository.")));
        }
        else {
        	try {
        		// Iterate all selected items.
        		$c = count($selrepos);
        		for($i = 0; $i < $c; $i++) {
        			$oR = new \SvnAdmin\Entities\Repository($selrepos[$i], $varParentIdentifier);

        			$b = $this->engine->getRepositoryEditProvider()->delete($oR);
        			if (!$b) {
        				throw new Exception(tr("Could not delete repository %0", array($oR->name)));
        			}
                	$this->engine->addMessage(tr("The repository %0 has been deleted.", array($oR->name)));

                	// Remove Access-Paths of the repository.
                	if ($remove_accesspaths)
                	{
                		try {
        					$apList = $this->engine->getAccessPathViewProvider()->getPathsOfRepository($oR);
        					foreach ($apList as $ap)
        					{
        						$this->engine->getAccessPathEditProvider()->deleteAccessPath($ap);
        						$this->engine->addMessage(tr('Removed Access-Path "%0"', array($ap->getPath())));
        					}
        					$this->engine->getAccessPathEditProvider()->save();
                		}
                		catch (Exception $ex2) {
                			$this->engine->addException($ex2);
                		}
                	} // if ($remove_accesspath)

        		}
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
