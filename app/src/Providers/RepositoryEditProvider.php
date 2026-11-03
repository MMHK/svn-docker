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

class RepositoryEditProvider implements \SvnAdmin\Contracts\IRepositoryEditProvider
{
	/**
	 * Object to handle operations of the "svn" executable.
	 * @var \SvnAdmin\Support\SvnClient
	 */
    private $_svnClient = NULL;

    /**
     * Object to handle operations of the "svnadmin" executable.
     * @var \SvnAdmin\Support\SvnAdmin
     */
	private $_svnAdmin = NULL;

	/**
	 * Holds the singelton instance of this class.
	 * @var SvnAdmin\Providers\RepositoryEditProvider
	 */
	private static $_instance = NULL;
	
	/**
	 * Holds multiple repository configurations.
	 * e.g.: array(
	 *			0 => array(
	 *				'SVNParentPath' => '/var/svn/repositories',
	 *				...
	 *			),
	 *			1 => array(
	 *				'SVNParentPath' => '/var/svn/repository-archive-2011',
	 *				...
	 *			),
	 *			2 => array(
	 *				'SVNParentPath' => '/var/svn/repository-archive-2012',
	 *				...
	 *			),
	 *		)
	 * @var array
	 */
	private $_config = array();
	
	/**
	 * Initializes the object by Engine configuration. 
	 */
	public function __construct()
	{
		$engine = \SvnAdmin\Core\Engine::getInstance();
		$config = $engine->getConfig();
		
		// Subversion class for browsing.
		$this->_svnClient = new \SvnAdmin\Support\SvnClient($engine->getConfig()
				->getValue('Repositories:svnclient', 'SvnExecutable'));
		
		// Subversion class for administration.
		$this->_svnAdmin = new \SvnAdmin\Support\SvnAdmin($engine->getConfig()
				->getValue('Repositories:svnclient', 'SvnAdminExecutable'));
		
		// Load default repository location configuration.
		$defaultSvnParentPath = $engine->getConfig()
				->getValue('Repositories:svnclient', 'SVNParentPath');
		
		// Set as default.
		$this->_config[0]['SVNParentPath'] = $defaultSvnParentPath;
		$this->_config[0]['description'] = 'Repositories';
		
		// Issue #5: Support multiple path values for SVNParentPath
		// Try to load more repository locations.
		$index = (int) 1;
		while (true) {
			$svnParentPath = $config->getValue('Repositories:svnclient:' . $index, 'SVNParentPath');
			if ($svnParentPath != null) {
				$this->_config[$index]['SVNParentPath'] = $svnParentPath;
			}
			else {
				break;
			}
			
			$description = $config->getValue('Repositories:svnclient:' . $index, 'Description');
			if ($description != null) {
				$this->_config[$index]['description'] = $description;
			}
			
			++$index;
		}
	}

	/**
	 * Gets the singelton instance of this object.
	 *
	 * @return SvnAdmin\Providers\RepositoryEditProvider
	 */
	public static function getInstance()
	{
		if (self::$_instance == NULL) {
			self::$_instance = new self();
		}
		return self::$_instance;
    }

    /**
     * (non-PHPdoc)
     * @see SvnAdmin\Contracts\IProvider::init()
     */
	public function init()
	{
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IEditProvider::save()
	 */
	public function save()
	{
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IRepositoryEditProvider::create()
	 */
	public function create(\SvnAdmin\Entities\Repository $oRepository)
	{
		$svnParentPath = $this->getRepositoryConfigValue($oRepository, 'SVNParentPath');
		
		if (!file_exists($svnParentPath)) {
			throw new \Exception("The repository parent path doesn't exists: " .
					$svnParentPath);
		}

		$path = $svnParentPath . '/' . $oRepository->name;
		$this->_svnAdmin->create($path);

		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IRepositoryEditProvider::delete()
	 */
	public function delete(\SvnAdmin\Entities\Repository $oRepository)
	{
		$svnParentPath = $this->getRepositoryConfigValue($oRepository, 'SVNParentPath');
		
		if ($svnParentPath == NULL) {
			throw new \Exception('Invalid parent-identifier: ' .
					$oRepository->getParentIdentifier());
		}
		
		$path = $svnParentPath . '/' . $oRepository->name;
		$this->_svnAdmin->delete($path);
		
		return true;
	}

	/**
	 * (non-PHPdoc)
	 * @see SvnAdmin\Contracts\IRepositoryEditProvider::mkdir()
	 */
	public function mkdir(\SvnAdmin\Entities\Repository $oRepository, array $paths)
	{
		$svnParentPath = $this->getRepositoryConfigValue($oRepository, 'SVNParentPath');
		
		if ($svnParentPath == NULL) {
			throw new \Exception('Invalid parent-identifier: ' .
					$oRepository->getParentIdentifier());
		}
		
		// Create absolute paths.
		for ($i = 0; $i < count($paths); ++$i) {
			$paths[$i] = $svnParentPath . '/' . $oRepository->name . '/' . $paths[$i];
		}
		
		$this->_svnClient->svn_mkdir($paths);
		
    	return true;
    }
	
	/**
	 * Gets the configuration value associated to the given Repository object
	 * (identified by 'parentIdentifier')
	 * 
	 * @param \SvnAdmin\Entities\Repository $oRepository
	 * @param string $key
	 * @return string
	 */
	protected function getRepositoryConfigValue(\SvnAdmin\Entities\Repository $oRepository, $key = 'SVNParentPath')
	{
		$v = null;

		if ($oRepository->parentIdentifier === null) {
			$v = $this->_config[0][$key];
		}
		else if (isset($this->_config[$oRepository->parentIdentifier])){
			$v = $this->_config[$oRepository->parentIdentifier][$key];
		}
		
		return $v;
	}
}
?>