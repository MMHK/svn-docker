<?php
namespace SvnAdmin\Actions;

class CreateRepository extends Action
{
    public function execute(): void
    {
        //
        // Authentication
        //

        if (!$this->engine->isProviderActive(PROVIDER_REPOSITORY_EDIT)) {
        	$this->engine->forwardError(ERROR_INVALID_MODULE);
        }

        $this->engine->checkUserAuthentication(true, ACL_MOD_REPO, ACL_ACTION_ADD);

        //
        // HTTP Request Vars
        //

        $varParentIdentifierEnc = get_request_var('pi');
        $reponame = get_request_var("reponame");

        $varParentIdentifier = rawurldecode($varParentIdentifierEnc);

        //
        // Validation
        //

        if ($reponame == NULL) {
        	$this->engine->addException(new ValidationException(tr("You have to fill out all fields.")));
        }
        else {
        	$r = new \SvnAdmin\Entities\Repository($reponame, $varParentIdentifier);

        	// Create repository.
        	try {
        		$this->engine->getRepositoryEditProvider()->create($r);
        		$this->engine->getRepositoryEditProvider()->save();
        		$this->engine->addMessage(tr("The repository %0 has been created successfully", array($reponame)));

        		// Create the access path now.
        		try {
        			if (get_request_var("accesspathcreate") != NULL
        				&& $this->engine->isProviderActive(PROVIDER_ACCESSPATH_EDIT)) {

        				$ap = new \SvnAdmin\Entities\AccessPath($reponame . ':/');

        				if ($this->engine->getAccessPathEditProvider()->createAccessPath($ap)) {
        					$this->engine->getAccessPathEditProvider()->save();
        				}
        			}
        		}
        		catch (Exception $e2) {
        			$this->engine->addException($e2);
        		}

        		// Create a initial repository structure.
        		try {
        			$repoPredefinedStructure = get_request_var("repostructuretype");
        			if ($repoPredefinedStructure != NULL) {

        				switch ($repoPredefinedStructure) {
        					case "single":
        						$this->engine->getRepositoryEditProvider()
        							->mkdir($r, array('trunk', 'branches', 'tags'));
        						break;

        					case "multi":
        						$projectName = get_request_var("projectname");
        						if ($projectName != NULL) {
        							$this->engine->getRepositoryEditProvider()
        								->mkdir($r, array(
        									$projectName . '/trunk',
        									$projectName . '/branches',
        									$projectName . '/tags'
        								));
        						}
        						else {
        							throw new ValidationException(tr("Missing project name"));
        						}
        						break;
        				}
        			}
        		}
        		catch (Exception $e3) {
        			$this->engine->addException($e3);
        		}
        	}
        	catch (Exception $e) {
        		$this->engine->addException($e);
        	}
        }
    }
}
