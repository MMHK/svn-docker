<?php
/**
 * iF.SVNAdmin
 * Copyright (c) 2010 by Manuel Freiholz
 * http://www.insanefactory.com/
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; version 2
 * of the License.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.
 */
// PHP 8 起 dynamic property / null 傳參等 deprecation 太吵，只留真正的錯誤。
error_reporting(E_ALL & ~E_DEPRECATED);
// Composer autoload：SvnAdmin\ 的類別，以及全局函式（src/Support/functions.php、
// src/Support/globals.php、src/Config/IniFileFunctions.php）與
// src/Legacy/compat.php（舊 FQCN 的 class_alias）。
require_once(__DIR__ . '/../vendor/autoload.php');
set_exception_handler('exception_handler');

// Check PHP version.
/*
if (!checkPHPVersion("5.3")) {
  echo "Wrong PHP version. The minimum required version is: 5.3";
  exit(1);
}
*/

// Does the config.ini file exists?
if (!file_exists("./data/config.ini"))
{
  if (!copy("./data/config.tpl.ini", "./data/config.ini"))
    throw new Exception("Could not copy configuration file template. Require write permission (777) to \"data\" folder and all containing files.");
  header("Location: settings.php");
  exit(0);
}

/**
 * Password encryption type for BASIC authentication.
 *
 * Optional, nothing is defined by default (the Apache APR1 MD5 hash is used):
 *   define("SVNADMIN_HTPASSWD_DEFAULT_CRYPT", "CRYPT"); // Unix only.
 *   define("SVNADMIN_HTPASSWD_DEFAULT_CRYPT", "SHA1");
 *   define("SVNADMIN_HTPASSWD_DEFAULT_CRYPT", "MD5"); // Custom Apache APR1 MD5 hash.
 */

/**
 * PHP/Subversion bug workarround for:
 * Custom config-directory for subversion.
 */
//define("IF_SVNBaseC_ConfigDir", "E:/Development/Test/temp svnadmin/svn-config-dir");

// The former iF.CoreLib2 classes (include/ifcorelib) are now PSR-4 classes below
// app/src and are loaded by the Composer autoloader.

/**
 * iF.SVNAdmin version.
 */
define("MAJOR_VERSION", "1");
define("MINOR_VERSION", "6.2");
define("VERSION_EXTRA", "");

/**
 * Constant ACL modules.
 */
define("ACL_MOD_BASIC",                "basics");
define("ACL_MOD_REPO",                 "repositories");
define("ACL_MOD_USER",                 "users");
define("ACL_MOD_GROUP",                "groups");
define("ACL_MOD_ACCESSPATH",           "accesspaths");
define("ACL_MOD_ROLE",                 "roles");
define("ACL_MOD_SETTINGS",             "settings");
define("ACL_MOD_PROJECTMANAGER",       "projectmanagers");

/**
 * Constant ACL actions.
 */
define("ACL_ACTION_VIEW",				"view");
define("ACL_ACTION_ADD",				"add");
define("ACL_ACTION_DELETE",				"delete");
define("ACL_ACTION_ASSIGN",				"assign");
define("ACL_ACTION_UNASSIGN",			"unassign");
define("ACL_ACTION_LOGIN",				"login");			// ACL_MOD_BASIC only!
define("ACL_ACTION_CHANGEPASS",			"changepass");		// ACL_MOD_USER only!
define("ACL_ACTION_CHANGEPASS_OTHER",	"changepassother");	// ACL_MOD_USER only!
define("ACL_ACTION_CHANGE",				"change");			// ACL_MOD_SETTINGS only (atm...)!

/*
 * Switch current locale procecure.
 */
$requestedLocale = get_request_var("locale");
if ($requestedLocale != null)
{
  $_COOKIE["locale"] = $requestedLocale;
  @setcookie("locale", $requestedLocale, time()+60*60*24*365); // 365 days
}

/**
 * The variable is a global variable and can be called in each class
 * and function which are used with this application.
 *
 * @var $appEngine \SvnAdmin\Core\Engine
 * @global $appEngine \SvnAdmin\Core\Engine
 * @deprecated No longer use this global variable!
 */
$appEngine = \SvnAdmin\Core\Engine::getInstance();

/**
 * Global object to translate strings into different languages.
 *
 * @global \SvnAdmin\I18n\Translator
 */
$appTR = \SvnAdmin\I18n\Translator::getInstance();
$appTR->setTranslationDirectory($appEngine->getConfig()->getValue("Translation", "Directory"));

/**
 * The template of the current opened page.
 * It supports translations and ACL's.
 *
 * @global \SvnAdmin\View\Template
 */
$appTemplate = new \SvnAdmin\View\Template;


$cfg = $appEngine->getConfig();

/**
 * User view provider.
 * This deployment uses the htpasswd file (passwd) only.
 */
if ($cfg->getValue("Engine:Providers", "UserViewProviderType") == "passwd")
{
  $userView = \SvnAdmin\Providers\PasswdUserProvider::getInstance();
  $appEngine->setUserViewProvider( $userView );
}

/**
 * User edit provider.
 */
if ($cfg->getValue("Engine:Providers", "UserEditProviderType") == "passwd")
{
  $userEdit = \SvnAdmin\Providers\PasswdUserProvider::getInstance();
  $appEngine->setUserEditProvider( $userEdit );
}

/**
 * Group view provider.
 */
if ($cfg->getValue("Engine:Providers", "GroupViewProviderType") == "svnauthfile")
{
  $groupView = \SvnAdmin\Providers\AuthFileGroupAndPathProvider::getInstance();
  $appEngine->setGroupViewProvider( $groupView );
}

/**
 * Group edit provider.
 */
if ($cfg->getValue("Engine:Providers", "GroupEditProviderType") == "svnauthfile" )
{
  $groupEdit = \SvnAdmin\Providers\AuthFileGroupAndPathProvider::getInstance();
  $appEngine->setGroupEditProvider( $groupEdit );
}

/**
 * Access-Path view provider.
 */
if ($cfg->getValue("Engine:Providers", "AccessPathViewProviderType") == "svnauthfile")
{
  $pathsView = \SvnAdmin\Providers\AuthFileGroupAndPathProvider::getInstance();
  $appEngine->setAccessPathViewProvider( $pathsView );
}

/**
 * Access-Path edit provider.
 */
if ($cfg->getValue("Engine:Providers", "AccessPathEditProviderType") == "svnauthfile")
{
  $pathsEdit = \SvnAdmin\Providers\AuthFileGroupAndPathProvider::getInstance();
  $appEngine->setAccessPathEditProvider( $pathsEdit );
}

/**
 * Repository view provider.
 */
if ($cfg->getValue("Engine:Providers", "RepositoryViewProviderType") == "svnclient")
{
  $repoView = \SvnAdmin\Providers\RepositoryViewProvider::getInstance();
  $appEngine->setRepositoryViewProvider( $repoView );
}

/**
 * Repository edit provider.
 */
if ($cfg->getValue("Engine:Providers", "RepositoryEditProviderType") == "svnclient")
{
  $repoEdit = \SvnAdmin\Providers\RepositoryEditProvider::getInstance();
  $appEngine->setRepositoryEditProvider( $repoEdit );
}

/**
 * Authentication status.
 */
if ($cfg->getValue("Engine:Providers", "AuthenticationStatus") == "basic")
{
  \SvnAdmin\Http\Session::instance()->start();
  $o = new \SvnAdmin\Providers\EngineBaseAuthenticator;
  $appEngine->setAuthenticator($o);

  // Engine: IAclManager
  // The ACL feature can only be used with authentication.
  if (true)
  {
    $o = new \SvnAdmin\Acl\FSAclManager($cfg->getValue("ACLManager", "UserRoleAssignmentFile"));
    $appEngine->setAclManager($o);
    $appTemplate->setAcl($appEngine);
  }
}

/**
 * An administrator role MUST be defined, if the authentication is active.
 * Otherwise nobody could setup the application.
 */
$appCurrentScriptFile = currentScriptFileName();
if ($appEngine->isAuthenticationActive() && $appCurrentScriptFile != "settings.php")
{
  if (!$appEngine->getAclManager()->hasAdminDefined())
  {
    header("Location: settings.php");
    exit(0);
  }
}

///////////////////////////////////////////////////////////////////////////////
// Global User Initinalizations.
///////////////////////////////////////////////////////////////////////////////

//
// Set user locale.
// Check whether the user has choosen a specific language
//

if (isset($_COOKIE["locale"]) && !empty($_COOKIE["locale"]))
{
	// Get locale from user cookie.
	$appTR->setCurrentLocale($_COOKIE["locale"]);
}
else
{
	// Fallback to default locale.
	$appTR->setCurrentLocale(
			$appEngine->getConfig()->getValue("GUI", "DefaultLocale", "en_US")
		);
}

///////////////////////////////////////////////////////////////////////////////
// Global Template Variables.
/////////////////////////////////////////////////////////////////////////////

//
// Locales
//

if (true)
{
	SetValue("LocaleList", $appTR->getAvailableLocales());
}

$appTR->loadModule("global");
?>
