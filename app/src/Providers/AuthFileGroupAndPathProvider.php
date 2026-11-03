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
namespace SvnAdmin\Providers;

class AuthFileGroupAndPathProvider implements	\SvnAdmin\Contracts\IGroupViewProvider,
												\SvnAdmin\Contracts\IGroupEditProvider,
												\SvnAdmin\Contracts\IPathsViewProvider,
												\SvnAdmin\Contracts\IPathsEditProvider
{
	/**
	 * The singelton instance of this class.
	 * @var \SvnAdmin\Providers\AuthFileGroupAndPathProvider
	 */
	private static $m_instance = NULL;

	/**
	 * @var bool
	 */
	private $m_init_done = false;

	/**
	 * The object to manage the SVNAuthFile.
	 * @var \SvnAdmin\Support\SvnAuthFile
	 */
	private $m_authfile = null;


	/**
	 * Constructor.
	 */
	public function __construct()
	{
	}

	/**
	 * Gets the singelton instance of this object.
	 *
	 * @return \SvnAdmin\Providers\AuthFileGroupAndPathProvider
	 */
	public static function getInstance()
	{
		if (self::$m_instance == null)
		{
			self::$m_instance = new AuthFileGroupAndPathProvider();
		}
		return self::$m_instance;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IProvider::init()
	 */
	public function init()
	{
		global $appEngine;
		if( !$this->m_init_done )
		{
			$this->m_init_done = true;
			$this->m_authfile = new \SvnAdmin\Support\SvnAuthFile($appEngine->getConfig()->getValue("Subversion", "SVNAuthFile"));
		}
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::reset()
	 */
	public function reset()
	{
		$E = \SvnAdmin\Core\Engine::getInstance();
		$this->m_authfile = new \SvnAdmin\Support\SvnAuthFile($E->getConfig()->getValue("Subversion", "SVNAuthFile"));
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IEditProvider::save()
	 */
	public function save()
	{
		return $this->m_authfile->save();
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IViewProvider::isUpdateable()
	 */
	public function isUpdateable()
	{
		return false;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IViewProvider::update()
	 */
	public function update()
	{
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupViewProvider::getGroups()
	 */
	public function getGroups()
	{
		$groupNamesArray = $this->m_authfile->groups();
		$retArray = array();
		if( is_array( $groupNamesArray ) )
		{
			for( $i=0; $i<count($groupNamesArray); $i++ )
			{
				$groupObj = new \SvnAdmin\Entities\Group;
				$groupObj->id = $groupNamesArray[$i];
				$groupObj->name = $groupNamesArray[$i];
				array_push( $retArray, $groupObj );
			}
		}
		return $retArray;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupViewProvider::groupExists()
	 */
	public function groupExists($objGroup)
	{
		return $this->m_authfile->groupExists($objGroup->name);
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupViewProvider::getGroupsOfUser()
	 */
	public function getGroupsOfUser($objUser)
	{
		$retArray = array();
		$groupNamesArray = $this->m_authfile->groupsOfUser( $objUser->name );

		if( is_array($groupNamesArray) )
		{
			for( $i=0; $i<count($groupNamesArray); $i++ )
			{
				$groupObj = new \SvnAdmin\Entities\Group;
				$groupObj->id = $groupNamesArray[$i];
				$groupObj->name = $groupNamesArray[$i];
				array_push( $retArray, $groupObj );
			}
		}
		return $retArray;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupViewProvider::getUsersOfGroup()
	 */
	public function getUsersOfGroup($objGroup)
    {
		$retArray = array();
		$userNamesArray = $this->m_authfile->usersOfGroup( $objGroup->name );

		if( is_array($userNamesArray) )
		{
			for( $i=0; $i<count($userNamesArray); $i++ )
			{
				$userObj = new \SvnAdmin\Entities\User();
				$userObj->id = $userNamesArray[$i];
				$userObj->name = $userNamesArray[$i];
				array_push( $retArray, $userObj );
			}
		}
		return $retArray;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupViewProvider::isUserInGroup()
	 */
	public function isUserInGroup($objUser, $objGroup)
	{
		return $this->m_authfile->isUserInGroup( $objGroup->name, $objUser->name );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupEditProvider::addGroup()
	 */
	public function addGroup( $objGroup )
	{
		return $this->m_authfile->createGroup($objGroup->name);
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupEditProvider::deleteGroup()
	 */
	public function deleteGroup( $objGroup )
	{
		return $this->m_authfile->deleteGroup($objGroup->name);
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupEditProvider::assignUserToGroup()
	 */
	public function assignUserToGroup( $objUser, $objGroup )
	{
		return $this->m_authfile->addUserToGroup( $objGroup->name, $objUser->name );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupEditProvider::removeUserFromGroup()
	 */
	public function removeUserFromGroup( $objUser, $objGroup )
	{
		return $this->m_authfile->removeUserFromGroup( $objUser->name, $objGroup->name );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IGroupEditProvider::removeUserFromAllGroups()
	 */
	public function removeUserFromAllGroups( $objUser )
	{
		$groups = $this->m_authfile->groupsOfUser($objUser->name);
		for( $i=0; $i<count($groups); $i++ )
		{
			$this->m_authfile->removeUserFromGroup( $objUser->name, $groups[$i] );
		}
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsViewProvider::getPaths()
	 */
	public function getPaths()
	{
		$list = array();
		$paths = $this->m_authfile->repositories();
		for( $i=0; $i<count($paths); $i++ )
		{
			$o = new \SvnAdmin\Entities\AccessPath;
			$o->path = $paths[$i];
			array_push( $list, $o );
		}
		return $list;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsViewProvider::getPathsOfGroup()
	 */
	public function getPathsOfGroup( $objGroup )
	{
		$list = array();
		$perms = $this->m_authfile->permissionsOfGroup( $objGroup->name );
		$permsLen = count($perms);
		for( $i=0; $i<$permsLen; $i++ )
		{
			$oAP = new \SvnAdmin\Entities\AccessPath;
			$oAP->path = $perms[$i][0];
			$oAP->perm = self::resolvePermission2($perms[$i][1]);
			array_push( $list, $oAP );
		}
		return $list;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsViewProvider::getPathsOfUser()
	 */
	public function getPathsOfUser( $objUser )
	{
		$list = array();
		$perms = $this->m_authfile->permissionsOfUser( $objUser->name );
		$permsLen = count($perms);
		for( $i=0; $i<$permsLen; $i++ )
		{
			$oAP = new \SvnAdmin\Entities\AccessPath;
			$oAP->path = $perms[$i][0];
			$oAP->perm = self::resolvePermission2($perms[$i][1]);
			$oAP->inherited = $perms[$i][2];
			array_push( $list, $oAP );
		}
		return $list;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsViewProvider::getPathsOfRepository()
	 */
	public function getPathsOfRepository($objRepository)
	{
		$list = array();
		$paths = $this->m_authfile->repositories();

		// Build root access path for repository. e.g.: "my_repo" => "my_repo:/"
		$repo_access_path_root = $objRepository->getName() . ':/';

		for( $i=0; $i<count($paths); $i++ )
		{
			$pos = strpos($paths[$i], $repo_access_path_root);
			if ($pos === 0)
			{
				$list[] = new \SvnAdmin\Entities\AccessPath($paths[$i]);
			}
		}

		return $list;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsViewProvider::getUsersOfPath()
	 */
	public function getUsersOfPath($objAccessPath)
	{
		$ret = array();

		$users = $this->m_authfile->usersOfRepository($objAccessPath->getPath());
		foreach ($users as $u)
		{
			$perm = $this->m_authfile->permissionsOfUser($u, false, $objAccessPath->getPath());
			$permString = $this->resolvePermission2($perm[0][1]);

			$o = new \SvnAdmin\Entities\User();
			$o->id = $u;
			$o->name = $u;
			$o->perm = $permString;
			$ret[] = $o;
		}

		return $ret;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsViewProvider::getGroupsOfPath()
	 */
	public function getGroupsOfPath( $objAccessPath )
	{
		$ret = array();

		$groups = $this->m_authfile->groupsOfRepository($objAccessPath->getPath());
		foreach ($groups as $g)
		{
			$perm = $this->m_authfile->permissionsOfGroup($g, false, $objAccessPath->getPath());
			$permString = $this->resolvePermission2($perm[0][1]);

			$o = new \SvnAdmin\Entities\Group;
			$o->id = $g;
			$o->name = $g;
			$o->perm = $permString;
			$ret[] = $o;
		}

		return $ret;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::deleteAccessPath()
	 */
	public function deleteAccessPath($objAccessPath)
	{
		return $this->m_authfile->removeRepositoryPath( $objAccessPath->path );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::createAccessPath()
	 */
	public function createAccessPath($objAccessPath)
	{
		return $this->m_authfile->addRepositoryPath( $objAccessPath->path );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::removeGroupFromAllAccessPaths()
	 */
	public function removeGroupFromAllAccessPaths($objGroup)
	{
		$paths = $this->m_authfile->repositoryPathsOfGroup( $objGroup->name );
		for( $i=0; $i<count($paths); $i++ )
		{
			$this->m_authfile->removeGroupFromRepository( $objGroup->name, $paths[$i] );
		}
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::removeUserFromAllAccessPaths()
	 */
	public function removeUserFromAllAccessPaths($objUser)
	{
		$paths = $this->m_authfile->repositoryPathsOfUser( $objUser->name );
		for( $i=0; $i<count($paths); $i++ )
		{
			$this->m_authfile->removeUserFromRepository( $objUser->name, $paths[$i] );
		}
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::removeUserFromAccessPath()
	 */
	public function removeUserFromAccessPath($objUser, $objAccessPath)
	{
		return $this->m_authfile->removeUserFromRepository( $objUser->name, $objAccessPath->path );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::removeGroupFromAccessPath()
	 */
	public function removeGroupFromAccessPath($objGroup, $objAccessPath)
	{
		return $this->m_authfile->removeGroupFromRepository( $objGroup->name, $objAccessPath->path );
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::assignGroupToAccessPath()
	 */
	public function assignGroupToAccessPath($objGroup, $objAccessPath, $objPermission)
	{
		$p = self::resolvePermission( $objPermission );
		if( $p !== FALSE )
		{
			return $this->m_authfile->addGroupToRepository( $objGroup->name, $objAccessPath->path, $p );
		}
		return false;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IPathsEditProvider::assignUserToAccessPath()
	 */
	public function assignUserToAccessPath($objUser, $objAccessPath, $objPermission)
	{
		$p = self::resolvePermission( $objPermission );
		if( $p !== FALSE )
		{
			return $this->m_authfile->addUserToRepository( $objUser->name, $objAccessPath->path, $p );
		}
		return false;
	}

	/**
	 *
	 * @param \SvnAdmin\Entities\Permission $objPermission
	 *
	 * @return string
	 */
	private function resolvePermission( $objPermission )
	{
		$p = $objPermission->getPerm();
		if( $p == \SvnAdmin\Entities\Permission::$PERM_NONE )
		{
			return \SvnAdmin\Support\SvnAuthFile::$PERMISSION_NONE;
		}
		else if( $p == \SvnAdmin\Entities\Permission::$PERM_READ )
		{
			return \SvnAdmin\Support\SvnAuthFile::$PERMISSION_READ;
		}
		else if( $p == \SvnAdmin\Entities\Permission::$PERM_READWRITE )
		{
			return \SvnAdmin\Support\SvnAuthFile::$PERMISSION_READWRITE;
		}
		return \SvnAdmin\Support\SvnAuthFile::$PERMISSION_NONE;
	}

	/**
	 *
	 * @param string $strPerm
	 *
	 * @return \SvnAdmin\Entities\Permission
	 */
	private function resolvePermission2( $strPerm )
	{
		if( $strPerm == \SvnAdmin\Support\SvnAuthFile::$PERMISSION_NONE )
		{
			return \SvnAdmin\Entities\Permission::$PERM_NONE;
		}
		else if( $strPerm == \SvnAdmin\Support\SvnAuthFile::$PERMISSION_READ )
		{
			return \SvnAdmin\Entities\Permission::$PERM_READ;
		}
		else if( $strPerm == \SvnAdmin\Support\SvnAuthFile::$PERMISSION_READWRITE )
		{
			return \SvnAdmin\Entities\Permission::$PERM_READWRITE;
		}
		return \SvnAdmin\Entities\Permission::$PERM_NONE;
	}
}
?>