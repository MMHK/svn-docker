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
require_once("include/config.inc.php");

if (!$appEngine->isAccessPathViewActive() || (!$appEngine->isUserViewActive() && !$appEngine->isGroupViewActive()))
{
	$appEngine->forwardInvalidModule(true);
}
$appEngine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN);
$appTR->loadModule("permissionassign");


// Form request.
$assign = check_request_var('assign');
if ($assign)
{
	$appEngine->handleAction('assign_usergrouptoaccesspath');
}

// Basic view data.
$users		= array();
$groups		= array();
$paths		= array();

if ($appEngine->isUserViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_USER, ACL_ACTION_VIEW))
{
	$users = $appEngine->getUserViewProvider()->getUsers();
	usort($users, array('\SvnAdmin\Entities\User',"compare"));
}

if ($appEngine->isGroupViewActive() && $appEngine->checkUserAuthentication(false, ACL_MOD_GROUP, ACL_ACTION_VIEW))
{
	$groups = $appEngine->getGroupViewProvider()->getGroups();
	usort($groups, array('\SvnAdmin\Entities\Group',"compare"));
}

if (true)
{
	$paths = $appEngine->getAccessPathViewProvider()->getPaths();

	// Filter access-paths for project-managers.
	if ($appEngine->isAuthenticationActive())
	{
		$currentUsername = $appEngine->getSessionUsername();
		if ($appEngine->getAclManager()->isUserAccessPathManager($currentUsername))
		{
			$paths = $appEngine->getAclManager()->filterAccessPathsList($currentUsername, $paths);
		}
	}

	usort($paths, array('\SvnAdmin\Entities\AccessPath',"compare") );
}

SetValue("PermNone", \SvnAdmin\Entities\Permission::$PERM_NONE);
SetValue("PermRead", \SvnAdmin\Entities\Permission::$PERM_READ);
SetValue("PermReadWrite", \SvnAdmin\Entities\Permission::$PERM_READWRITE);
SetValue("UserList", $users);
SetValue("GroupList", $groups);
SetValue("AccessPathList", $paths);
ProcessTemplate("permission/permissionassign.html.php");
?>