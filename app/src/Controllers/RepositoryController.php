<?php
namespace SvnAdmin\Controllers;

class RepositoryController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        return $this->list();
    }
    
    public function list(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isProviderActive(PROVIDER_REPOSITORY_VIEW)) {
            $appEngine->forwardError(ERROR_INVALID_MODULE);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_REPO, ACL_ACTION_VIEW);
        $appTR->loadModule("repositorylist");
        
        // Actions.
        if (check_request_var("delete")) {
            $appEngine->handleAction("delete_repository");
        } elseif (check_request_var('load')) {
            return $this->response;
        }
        
        // View data.
        $repositoryParentList = array();
        $repositoryList = array();
        try {
            $repositoryParentList = $appEngine->getRepositoryViewProvider()->getRepositoryParents();
            
            foreach ($repositoryParentList as $rp) {
                $repositoryList[$rp->identifier] = $appEngine->getRepositoryViewProvider()->getRepositoriesOfParent($rp);
                usort($repositoryList[$rp->identifier], array('\SvnAdmin\Entities\Repository', 'compare'));
            }
        } catch (\Exception $ex) {
            $appEngine->addException($ex);
        }
        
        $this->setValue('RepositoryParentList', $repositoryParentList);
        $this->setValue('RepositoryList', $repositoryList);
        $this->setValue('ShowDeleteButton', $appEngine->getConfig()->getValueAsBoolean('GUI', 'RepositoryDeleteEnabled', true));
        $this->processTemplate('repository/repositorylist.html.php');
        
        return $this->response;
    }
    
    public function create(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isProviderActive(PROVIDER_REPOSITORY_EDIT)) {
            $appEngine->forwardInvalidModule(true);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_REPO, ACL_ACTION_ADD);
        $appTR->loadModule("repositorycreate");
        
        // Actions.
        if (check_request_var('create')) {
            $appEngine->handleAction('create_repository');
        }
        
        $this->setValue('RepositoryParentList', $appEngine->getRepositoryViewProvider()->getRepositoryParents());
        $this->processTemplate("repository/repositorycreate.html.php");
        
        return $this->response;
    }
    
    public function view(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        if (!$appEngine->isProviderActive(PROVIDER_REPOSITORY_VIEW)) {
            $appEngine->forwardError(ERROR_INVALID_MODULE);
        }
        
        $appEngine->checkUserAuthentication(true, ACL_MOD_REPO, ACL_ACTION_VIEW);
        $appTR->loadModule("repositoryview");
        
        // HTTP Request Vars.
        $varParentIdentifierEnc = get_request_var('pi');
        $varRepoEnc = get_request_var('r');
        $varPathEnc = get_request_var('p');
        
        $varParentIdentifier = rawurldecode($varParentIdentifierEnc);
        $varRepo = rawurldecode($varRepoEnc);
        $varPath = rawurldecode($varPathEnc);
        
        $oR = new \SvnAdmin\Entities\Repository($varRepo, $varParentIdentifier);
        
        try {
            $repoPathList = $appEngine->getRepositoryViewProvider()->listPath($oR, $varPath);
            
            $apacheWebLink = $appEngine->getConfig()->getValue("GUI", "ApacheDirectoryListing");
            $customWebLink = $appEngine->getConfig()->getValue("GUI", "CustomDirectoryListing");
            $hasApacheWebLink = !empty($apacheWebLink) ? true : false;
            $hasCustomWebLink = !empty($customWebLink) ? true : false;
            
            $isRepositoryRoot = false;
            if ($varPath == null || $varPath == "/") {
                $isRepositoryRoot = true;
            }
            
            $itemList = array();
            foreach ($repoPathList as &$val) {
                if ($hasApacheWebLink || $hasCustomWebLink) {
                    $args = array($oR->getEncodedName(), $val->getEncodedRelativePath());
                    
                    if ($hasApacheWebLink) {
                        $val->apacheWebLink = \SvnAdmin\Support\StringUtils::arguments($apacheWebLink, $args);
                    }
                    
                    if ($hasCustomWebLink) {
                        $val->customWebLink = \SvnAdmin\Support\StringUtils::arguments($customWebLink, $args);
                    }
                }
                $itemList[] = $val;
            }
            
            $backLinkPath = "/";
            if (empty($varPath)) {
                $varPath = "";
            } else {
                $pos = strrpos($varPath, "/");
                if ($pos !== false && $pos > 0) {
                    $backLinkPath = substr($varPath, 0, $pos);
                }
            }
            
            $this->setValue("ApacheWebLink", $hasApacheWebLink);
            $this->setValue("CustomWebLink", $hasCustomWebLink);
            $this->setValue("ItemList", $itemList);
            $this->setValue("Repository", $oR);
            $this->setValue("BackLinkPath", $backLinkPath);
            $this->setValue("BackLinkPathEncoded", rawurlencode($backLinkPath));
            $this->setValue("CurrentPath", $varPath);
            $this->setValue("RepositoryRoot", $isRepositoryRoot);
        } catch (\Exception $ex) {
            $appEngine->addException($ex);
        }
        $this->processTemplate("repository/repositoryview.html.php");
        
        return $this->response;
    }
}
