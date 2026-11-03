<?php
namespace SvnAdmin\Http;

/**
 * Simple session service for the application.
 *
 * The session key is kept as "svnadmin_username" on purpose, so that sessions
 * created by older versions of the application stay valid.
 *
 * This class deliberately uses a static accessor instead of the container: the
 * legacy pages and actions are not wired through a dependency injection
 * container yet.
 */
class Session
{
	/**
	 * Name of the session key which holds the login name.
	 */
	public const USERNAME_KEY = "svnadmin_username";

	/**
	 * The singleton instance.
	 * @var Session|null
	 */
	private static $instance = null;

	/**
	 * Gets the singleton instance.
	 *
	 * @return Session
	 */
	public static function instance(): Session
	{
		if (self::$instance === null)
		{
			self::$instance = new Session();
		}
		return self::$instance;
	}

	/**
	 * Starts the PHP session, if it is not already active.
	 *
	 * The session settings (e.g. session.save_path) are taken from php.ini.
	 *
	 * @return void
	 */
	public function start(): void
	{
		if (session_status() !== PHP_SESSION_ACTIVE)
		{
			session_start();
		}
	}

	/**
	 * Gets the name of the logged in user, or NULL if nobody is logged in.
	 *
	 * @return string|null
	 */
	public function getUsername(): ?string
	{
		if (isset($_SESSION[self::USERNAME_KEY]) && $_SESSION[self::USERNAME_KEY] !== "")
		{
			return (string) $_SESSION[self::USERNAME_KEY];
		}
		return null;
	}

	/**
	 * Marks the given user as logged in. A NULL user logs the user out.
	 *
	 * @param string|null $username
	 * @return void
	 */
	public function setUsername(?string $username): void
	{
		if ($username === null)
		{
			$this->clear();
			return;
		}
		$_SESSION[self::USERNAME_KEY] = $username;
	}

	/**
	 * Removes the logged in user from the session.
	 *
	 * @return void
	 */
	public function clear(): void
	{
		unset($_SESSION[self::USERNAME_KEY]);
	}

	/**
	 * Destroys the whole session.
	 *
	 * @return void
	 */
	public function destroy(): void
	{
		$this->clear();
		if (session_status() === PHP_SESSION_ACTIVE)
		{
			session_destroy();
		}
	}
}
