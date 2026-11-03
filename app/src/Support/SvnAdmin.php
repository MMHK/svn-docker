<?php
namespace SvnAdmin\Support;

/**
 * Provides functionality of the "svnadmin" executable by using the
 * executable and parsing the output.
 *
 * @author Manuel Freiholz, insaneFactory
 */
class SvnAdmin extends SvnBase
{
	/**
	 * Path to the "svnadmin" binary.
	 * @var string
	 */
	private $m_svnadmin = NULL;


	/**
	 * Constructor.
	 *
	 * @param string $svn_admin_binary Absolute path to "svnadmin" executable.
	 * @throws SvnException
	 */
	public function __construct($svn_admin_binary)
	{
		parent::__construct();
		$this->trust_server_cert = false;
		$this->non_interactive = false;
		$this->m_svnadmin = $svn_admin_binary;

		// if (!file_exists($svn_admin_binary))
		// 	throw new SvnException('Path to "svnadmin" binary does not exist: '.$this->m_svnadmin);
		// if (!is_executable($svn_admin_binary))
		// 	throw new SvnException('Permission denied! Can not execute "svnadmin" executable: '.$this->m_svnadmin);
	}

	/**
	 * Creates a new empty repository.
	 *
	 * The repository type is fixed to "fsfs"; Subversion 1.9+ dropped support
	 * for the old Berkeley DB backend.
	 *
	 * @param string $path Absolute path to the new repository
	 * @throws SvnException
	 * @throws SvnCommandExecutionException
	 */
	public function create($path)
	{
		if (empty($path))
		{
			throw new SvnException('Empty path parameter for create() command.');
		}

		// Validate repository name.
		$pattern = '/^([a-z0-9\_\-]+)$/i';
		$repo_name = basename($path);

		if (!preg_match($pattern, $repo_name))
		{
			throw new SvnException('Invalid repository name: '.$repo_name.' (Allowed pattern: '.$pattern.')');
		}

		$args = array();
		if (!empty($this->config_directory))
		{
			$args["--config-dir"] = escapeshellarg($this->config_directory);
		}

		$args["--fs-type"] = escapeshellarg("fsfs");

		$cmd = self::create_svn_command($this->m_svnadmin, "create", self::encode_local_path($path), $args, false);

		$output = null;
		$exitCode = 0;
		exec($cmd, $output, $exitCode);

		if ($exitCode != 0)
		{
			throw new SvnCommandExecutionException('Command='.$cmd.'; Return='.$exitCode.'; Output='.$output.';');
		}
	}

	/**
	 * Deletes the repository at the given path.
	 *
	 * @param string $path Path to the repository.
	 * @return bool
	 */
	public function delete($path)
	{
		$files = glob($path."/*"/*, GLOB_MARK*/); // GLOB_MARK = Adds a ending slash to directory paths.
		foreach( $files as $f )
		{
			if (is_dir($f))
			{
				self::delete( $f );
			}
			else
			{
				chmod($f, 0777);
				unlink($f);
			}
		}

		if (is_dir($path))
		{
			rmdir($path);
		}

		return true;
	}
}
