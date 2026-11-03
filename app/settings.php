<?php
include("include/config.inc.php");

/**
 * The configuration of the web application.
 * @global $cfgEngine \SvnAdmin\Config\Config
 */
$cfgEngine = $appEngine->getConfig();

/**
 * Indicates whether the "setadmin" page should be shown.
 * @global $show_setadmin bool
 */
$show_setadmin = false;

/**
 * Indicates whether an user with the administrator role exists.
 * @global $hasAdminDefined bool
 */
$hasAdminDefined = true;
if ($appEngine->isAclManagerActive())
{
	$hasAdminDefined = $appEngine->getAclManager()->hasAdminDefined();
}

/**
 * Indicates whether it is the first start of the application.
 * @global $isFirstStart bool
 */
$isFirstStart = ($cfgEngine->getValue("Common", "FirstStart") == "1") ? true : false;

$skip_auth = false;
if (!$isFirstStart && !$hasAdminDefined)
{
	$skip_auth = true;
	$show_setadmin = true;
}

if (!$hasAdminDefined || $isFirstStart)
{
	$skip_auth = true;
}

// Check auth?
if (!$skip_auth)
{
	$appEngine->checkUserAuthentication(true, ACL_MOD_SETTINGS, ACL_ACTION_CHANGE);
}

$appTR->loadModule("settings");

////////////////////////////////////////////////////////////////////////////////
// Reset first start up value.
////////////////////////////////////////////////////////////////////////////////

if (check_request_var("firststart"))
{
  $cfgEngine->setValue("Common", "FirstStart", 1);
  $cfgEngine->saveToFile();

  header("Location: settings.php");
  exit(0);
}

////////////////////////////////////////////////////////////////////////////////
// Set administrator.
////////////////////////////////////////////////////////////////////////////////

if (check_request_var("setadmin") || $show_setadmin)
{
  // Assign the Administrator role.
  if (check_request_var("saveadmin"))
  {
    $selusers = get_request_var("selected_users");

    $oU = new \SvnAdmin\Entities\User;
    $oU->name = $selusers[0];

    $oR = new \SvnAdmin\Entities\Role;
    $oR->name = "Administrator";

    // Set selected user as Administrator.
	try {
		$b = $appEngine->getAclManager()->assignUserToRole($oU, $oR);
		if (!$b) {
			throw new Exception("ERROR: Can not assign user to role.");
		}

		$b = $appEngine->getAclManager()->save();
		if (!$b) {
			throw new Exception("ERROR: Can not save ACL changes.");
		}

		$appEngine->addMessage(tr("The user has been defined as admin. You can <a href=\"login.php\">login</a> now."));

      	header("Location: index.php");
		exit(0);
	}
	catch (Exception $except) {
		$appEngine->addException($except);
	}
  }

  // Display users, which can be defined as Administrator.
  if ($appEngine->isUserViewActive())
  {
    $users = $appEngine->getUserViewProvider()->getUsers(false);
    $usersCount = count($users);

    // If there are no users, we create one - if possible.
    if (empty($users))
    {
      if ($appEngine->isUserEditActive())
      {
        $u = new \SvnAdmin\Entities\User();
        $u->name = "admin";
        $u->password = "admin";

        try {
        	if (!$appEngine->getUserEditProvider()->addUser($u)) {
        		throw new Exception("ERROR: Can not add user.");
        	}

        	if (!$appEngine->getUserEditProvider()->save()) {
        		throw new Exception("ERROR: Can not save changes.");
        	}
        }
        catch (Exception $e) {
        	$appEngine->addException($e);
        }

        $r = new \SvnAdmin\Entities\Role;
        $r->name = "Administrator";

        try {
        	if (!$appEngine->getAclManager()->assignUserToRole($u, $r)) {
        		throw new Exception("ERROR: Can not assign Administrator role.");
        	}

        	if (!$appEngine->getAclManager()->save()) {
        		throw new Exception("ERROR: Can not save changes.");
        	}
        }
        catch (Exception $e) {
			$appEngine->addException($e);
        }

		SetValue("DefaultUserCreated", true);
      }
      else
      {
        // Display message. That no user edit provider is defined to create a
        // default admin user.
        SetValue("NoUserEditActive", true);
      }
    }
    // Display user selection.
    else
    {
      usort($users, array('\SvnAdmin\Entities\User',"compare"));
      SetValue("UserList", $users);
      SetValue("ShowUserSelection", true);
    }
  }
  else
  {
  	$appEngine->addException(new Exception(tr("You have to define a user view provider.")));
  }

  ProcessTemplate("settings/setadmin.html.php");
  exit(0);
}

////////////////////////////////////////////////////////////////////////////////
// System information (read only).
////////////////////////////////////////////////////////////////////////////////

/**
 * Describes the access state of a configured file or directory.
 *
 * @param string $path Absolute path from the configuration.
 * @return string Translated state of the path.
 */
function settings_path_state($path)
{
	if ($path === null || trim($path) === "")
	{
		return tr("Not configured");
	}

	if (!file_exists($path))
	{
		return tr("Does not exist");
	}

	$readable = is_readable($path);
	$writable = is_writable($path);

	if ($readable && $writable)
		return tr("Readable and writable");
	if ($readable)
		return tr("Readable, not writable");
	if ($writable)
		return tr("Not readable, writable");
	return tr("Not readable, not writable");
}

/**
 * Gets the version string of the configured "svn" executable.
 *
 * @param string $executable Path of the "svn" binary.
 * @return string Version or "-" if it could not be determined.
 */
function settings_svn_version($executable)
{
	if ($executable === null || trim($executable) === "")
	{
		$executable = "svn";
	}

	$output = array();
	$returnCode = 0;
	@exec(escapeshellarg($executable) . " --version --quiet 2>&1", $output, $returnCode);

	if ($returnCode === 0 && count($output) > 0)
	{
		return trim($output[0]);
	}

	return "-";
}

// Application version.
SetValue("AppVersion", MAJOR_VERSION . "." . MINOR_VERSION . (VERSION_EXTRA != "" ? "-" . VERSION_EXTRA : ""));

// PHP version.
SetValue("PHPVersion", PHP_VERSION);

// Subversion version.
$svnExecutable = $cfgEngine->getValue("Repositories:svnclient", "SvnExecutable");
SetValue("SvnVersion", settings_svn_version($svnExecutable));

// Subversion paths and their access state.
$svnAuthFile = $cfgEngine->getValue("Subversion", "SVNAuthFile");
SetValue("SVNAuthFile", $svnAuthFile);
SetValue("SVNAuthFileState", settings_path_state($svnAuthFile));

$svnUserFile = $cfgEngine->getValue("Users:passwd", "SVNUserFile");
SetValue("SVNUserFile", $svnUserFile);
SetValue("SVNUserFileState", settings_path_state($svnUserFile));

$svnParentPath = $cfgEngine->getValue("Repositories:svnclient", "SVNParentPath");
SetValue("SVNParentPath", $svnParentPath);
SetValue("SVNParentPathState", settings_path_state($svnParentPath));

// Active providers.
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
SetValue("ProviderTypes", $providerTypes);

// Process template.
ProcessTemplate("settings/backend.html.php");
?>
