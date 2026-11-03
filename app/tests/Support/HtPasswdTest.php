<?php
namespace SvnAdmin\Tests\Support;

use PHPUnit\Framework\TestCase;
use SvnAdmin\Support\HtPasswd;

/**
 * Pure logic tests for {@see HtPasswd}: no Apache, no SVN binaries, no DB.
 */
final class HtPasswdTest extends TestCase
{
	/** @var string */
	private $file;

	protected function setUp(): void
	{
		$this->file = tempnam(sys_get_temp_dir(), 'htpasswd');
	}

	protected function tearDown(): void
	{
		if (is_file($this->file)) {
			unlink($this->file);
		}
	}

	public function testCreateUserWritesApr1Hash(): void
	{
		$htpasswd = new HtPasswd($this->file);
		$this->assertTrue($htpasswd->createUser('alice', 'secret'));
		$this->assertTrue($htpasswd->writeToFile());

		$hash = $this->hashFor('alice');
		$this->assertMatchesRegularExpression(
			'/^\$apr1\$[.\/0-9A-Za-z]{8}\$[.\/0-9A-Za-z]{22}$/',
			$hash
		);
	}

	public function testHashesUseRandomSalt(): void
	{
		$htpasswd = new HtPasswd($this->file);
		$htpasswd->createUser('user1', 'same-password');
		$htpasswd->createUser('user2', 'same-password');
		$htpasswd->writeToFile();

		$hash1 = $this->hashFor('user1');
		$hash2 = $this->hashFor('user2');

		$this->assertNotSame($hash1, $hash2);
		$this->assertNotSame($this->saltOf($hash1), $this->saltOf($hash2));
	}

	public function testAuthenticateAcceptsCorrectPassword(): void
	{
		$htpasswd = new HtPasswd($this->file);
		$htpasswd->createUser('bob', 'correct horse');
		$htpasswd->writeToFile();

		$reloaded = new HtPasswd($this->file);
		$this->assertTrue($reloaded->init());
		$this->assertSame(array('bob'), $reloaded->getUserList());

		$this->assertTrue($reloaded->authenticate('bob', 'correct horse'));
		$this->assertFalse($reloaded->authenticate('bob', 'wrong password'));
		$this->assertFalse($reloaded->authenticate('nobody', 'correct horse'));
	}

	public function testUserExistsAndDeleteUser(): void
	{
		$htpasswd = new HtPasswd($this->file);
		$htpasswd->createUser('carol', 'pw');

		$this->assertTrue($htpasswd->userExists('carol'));
		$this->assertFalse($htpasswd->userExists('dave'));

		$this->assertTrue($htpasswd->deleteUser('carol'));
		$this->assertFalse($htpasswd->userExists('carol'));
		$this->assertFalse($htpasswd->deleteUser('carol'));
		$this->assertSame(11, $htpasswd->errno());
	}

	public function testAuthenticateSupportsPlainShaEntries(): void
	{
		// {SHA} entries are the second supported format of the auth file.
		$sha = '{SHA}' . base64_encode(sha1('sha-password', true));
		file_put_contents($this->file, "sha-user:" . $sha . "\n");

		$htpasswd = new HtPasswd($this->file);
		$this->assertTrue($htpasswd->init());
		$this->assertTrue($htpasswd->authenticate('sha-user', 'sha-password'));
		$this->assertFalse($htpasswd->authenticate('sha-user', 'nope'));
	}

	public function testInitFailsForMissingFile(): void
	{
		$htpasswd = new HtPasswd($this->file . '.missing');
		$this->assertFalse($htpasswd->init());
		$this->assertSame(1, $htpasswd->errno());
	}

	/**
	 * Reads the stored hash of the given user from the auth file.
	 */
	private function hashFor(string $username): string
	{
		foreach (file($this->file, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) as $line) {
			$parts = explode(':', $line, 2);
			if ($parts[0] === $username) {
				return $parts[1];
			}
		}
		$this->fail('No hash stored for user ' . $username);
	}

	/**
	 * Extracts the salt part of an APR1 hash ($apr1$<salt>$<hash>).
	 */
	private function saltOf(string $hash): string
	{
		$parts = explode('$', $hash);
		return $parts[2] ?? '';
	}
}
