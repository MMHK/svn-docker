<?php
namespace SvnAdmin\Controllers;

class SettingsController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $cfgEngine = $appEngine->getConfig();
        
        $show_setadmin = false;
        $hasAdminDefined = true;
        if ($appEngine->isAclManagerActive()) {
            $hasAdminDefined = $appEngine->getAclManager()->hasAdminDefined();
        }
        
        $isFirstStart = ($cfgEngine->getValue("Common", "FirstStart") == "1") ? true : false;
        
        $skip_auth = false;
        if (!$isFirstStart && !$hasAdminDefined) {
            $skip_auth = true;
            $show_setadmin = true;
        }
        
        if (!$hasAdminDefined || $isFirstStart) {
            $skip_auth = true;
        }
        
        if (!$skip_auth) {
            $appEngine->checkUserAuthentication(true, ACL_MOD_SETTINGS, ACL_ACTION_CHANGE);
        }
        
        $appTR->loadModule("settings");
        
        // Reset first start up value.
        if (check_request_var("firststart")) {
            $cfgEngine->setValue("Common", "FirstStart", 1);
            $cfgEngine->saveToFile();
            
            return $this->redirect("settings.php");
        }
        
        // Set administrator.
        if (check_request_var("setadmin") || $show_setadmin) {
            if (check_request_var("saveadmin")) {
                $selusers = get_request_var("selected_users");
                
                $oU = new \SvnAdmin\Entities\User();
                $oU->name = $selusers[0];
                
                $oR = new \SvnAdmin\Entities\Role();
                $oR->name = "Administrator";
                
                try {
                    $b = $appEngine->getAclManager()->assignUserToRole($oU, $oR);
                    if (!$b) {
                        throw new \Exception("ERROR: Can not assign user to role.");
                    }
                    
                    $b = $appEngine->getAclManager()->save();
                    if (!$b) {
                        throw new \Exception("ERROR: Can not save ACL changes.");
                    }
                    
                    $appEngine->addMessage(tr("The user has been defined as admin. You can <a href=\"login.php\">login</a> now."));
                    
                    return $this->redirect("index.php");
                } catch (\Exception $except) {
                    $appEngine->addException($except);
                }
            }
            
            if ($appEngine->isUserViewActive()) {
                $users = $appEngine->getUserViewProvider()->getUsers(false);
                
                if (empty($users)) {
                    if ($appEngine->isUserEditActive()) {
                        $u = new \SvnAdmin\Entities\User();
                        $u->name = "admin";
                        $u->password = "admin";
                        
                        try {
                            if (!$appEngine->getUserEditProvider()->addUser($u)) {
                                throw new \Exception("ERROR: Can not add user.");
                            }
                            
                            if (!$appEngine->getUserEditProvider()->save()) {
                                throw new \Exception("ERROR: Can not save changes.");
                            }
                        } catch (\Exception $e) {
                            $appEngine->addException($e);
                        }
                        
                        $r = new \SvnAdmin\Entities\Role();
                        $r->name = "Administrator";
                        
                        try {
                            if (!$appEngine->getAclManager()->assignUserToRole($u, $r)) {
                                throw new \Exception("ERROR: Can not assign Administrator role.");
                            }
                            
                            if (!$appEngine->getAclManager()->save()) {
                                throw new \Exception("ERROR: Can not save changes.");
                            }
                        } catch (\Exception $e) {
                            $appEngine->addException($e);
                        }
                        
                        $this->setValue("DefaultUserCreated", true);
                    } else {
                        $this->setValue("NoUserEditActive", true);
                    }
                } else {
                    usort($users, array('\SvnAdmin\Entities\User', "compare"));
                    $this->setValue("UserList", $users);
                    $this->setValue("ShowUserSelection", true);
                }
            } else {
                $appEngine->addException(new \Exception(tr("You have to define a user view provider.")));
            }
            
            $this->processTemplate("settings/setadmin.html.php");
            return $this->response;
        }
        
        // System information.
        $this->setValue("AppVersion", MAJOR_VERSION . "." . MINOR_VERSION . (VERSION_EXTRA != "" ? "-" . VERSION_EXTRA : ""));
        $this->setValue("PHPVersion", PHP_VERSION);
        
        $svnExecutable = $cfgEngine->getValue("Repositories:svnclient", "SvnExecutable");
        $this->setValue("SvnVersion", $this->svnVersion($svnExecutable));
        
        $svnAuthFile = $cfgEngine->getValue("Subversion", "SVNAuthFile");
        $this->setValue("SVNAuthFile", $svnAuthFile);
        $this->setValue("SVNAuthFileState", $this->pathState($svnAuthFile));
        
        $svnUserFile = $cfgEngine->getValue("Users:passwd", "SVNUserFile");
        $this->setValue("SVNUserFile", $svnUserFile);
        $this->setValue("SVNUserFileState", $this->pathState($svnUserFile));
        
        $svnParentPath = $cfgEngine->getValue("Repositories:svnclient", "SVNParentPath");
        $this->setValue("SVNParentPath", $svnParentPath);
        $this->setValue("SVNParentPathState", $this->pathState($svnParentPath));
        
        $providerTypes = array(
            "UserViewProviderType"        => $cfgEngine->getValue("Engine:Providers", "UserViewProviderType"),
            "UserEditProviderType"        => $cfgEngine->getValue("Engine:Providers", "UserEditProviderType"),
            "GroupViewProviderType"       => $cfgEngine->getValue("Engine:Providers", "GroupViewProviderType"),
            "GroupEditProviderType"       => $cfgEngine->getValue("Engine:Providers", "GroupEditProviderType"),
            "AccessPathViewProviderType"  => $cfgEngine->getValue("Engine:Providers", "AccessPathViewProviderType"),
            "AccessPathEditProviderType"  => $cfgEngine->getValue("Engine:Providers", "AccessPathEditProviderType"),
            "RepositoryViewProviderType"  => $cfgEngine->getValue("Engine:Providers", "RepositoryViewProviderType"),
            "RepositoryEditProviderType"  => $cfgEngine->getValue("Engine:Providers", "RepositoryEditProviderType")
        );
        $this->setValue("ProviderTypes", $providerTypes);
        
        $this->processTemplate("settings/backend.html.php");
        
        return $this->response;
    }
    
    private function pathState($path): string
    {
        if ($path === null || trim($path) === "") {
            return tr("Not configured");
        }
        
        if (!file_exists($path)) {
            return tr("Does not exist");
        }
        
        $readable = is_readable($path);
        $writable = is_writable($path);
        
        if ($readable && $writable) return tr("Readable and writable");
        if ($readable) return tr("Readable, not writable");
        if ($writable) return tr("Not readable, writable");
        return tr("Not readable, not writable");
    }
    
    private function svnVersion($executable): string
    {
        if ($executable === null || trim($executable) === "") {
            $executable = "svn";
        }
        
        $output = array();
        $returnCode = 0;
        @exec(escapeshellarg($executable) . " --version --quiet 2>&1", $output, $returnCode);
        
        if ($returnCode === 0 && count($output) > 0) {
            return trim($output[0]);
        }
        
        return "-";
    }
}
