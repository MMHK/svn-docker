<?php
/**
 * ACL Constants and version info for iF.SVNAdmin
 */

// Version constants (from original config.inc.php)
define("MAJOR_VERSION", "2");
define("MINOR_VERSION", "0.0");
define("VERSION_EXTRA", "");

// ACL Modules
define("ACL_MOD_BASIC",                "basics");
define("ACL_MOD_REPO",                 "repositories");
define("ACL_MOD_USER",                 "users");
define("ACL_MOD_GROUP",                "groups");
define("ACL_MOD_ACCESSPATH",           "accesspaths");
define("ACL_MOD_ROLE",                 "roles");
define("ACL_MOD_SETTINGS",             "settings");
define("ACL_MOD_PROJECTMANAGER",       "projectmanagers");

// ACL Actions
define("ACL_ACTION_VIEW",              "view");
define("ACL_ACTION_ADD",               "add");
define("ACL_ACTION_DELETE",            "delete");
define("ACL_ACTION_ASSIGN",            "assign");
define("ACL_ACTION_UNASSIGN",          "unassign");
define("ACL_ACTION_LOGIN",             "login");              // ACL_MOD_BASIC only!
define("ACL_ACTION_CHANGEPASS",        "changepass");         // ACL_MOD_USER only!
define("ACL_ACTION_CHANGEPASS_OTHER",  "changepassother");    // ACL_MOD_USER only!
define("ACL_ACTION_CHANGE",            "change");             // ACL_MOD_SETTINGS only (atm...)!

// Page constants
define("PAGE_HOME", "index.php");
define("PAGE_LOGIN", "login.php");
define("PAGE_ERROR", "error.php");

// Error types
define("ERROR_NO_ACCESS", "no_access");
define("ERROR_INVALID_MODULE", "invalid_module");

// Provider IDs
define("PROVIDER_USER_VIEW", 1);
define("PROVIDER_USER_EDIT", 2);
define("PROVIDER_GROUP_VIEW", 3);
define("PROVIDER_GROUP_EDIT", 4);
define("PROVIDER_ACCESSPATH_VIEW", 5);
define("PROVIDER_ACCESSPATH_EDIT", 6);
define("PROVIDER_REPOSITORY_VIEW", 7);
define("PROVIDER_REPOSITORY_EDIT", 8);
define("PROVIDER_AUTHENTICATION", 9);
