<?php
namespace SvnAdmin\Config;

/**
 * Creates {@see Config} objects for the application.
 *
 * Keeps the construction of the configuration service in one place, so that
 * the callers do not depend on the file handling of the class itself.
 */
class ConfigFactory
{
	/**
	 * Creates a new configuration object for the given file.
	 *
	 * @param string $configFile Path to the ini file.
	 * @return Config
	 */
	public static function create(string $configFile): Config
	{
		return new Config($configFile);
	}
}
