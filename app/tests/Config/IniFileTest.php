<?php
namespace SvnAdmin\Tests\Config;

use PHPUnit\Framework\TestCase;
use SvnAdmin\Config\IniFile;

/**
 * Tests the application's own INI dialect (if_parse_ini_file / {@see IniFile}),
 * which deliberately differs from PHP's native parse_ini_file().
 */
final class IniFileTest extends TestCase
{
	/** @var string */
	private $file;

	/** @var string */
	private $copy;

	protected function setUp(): void
	{
		$this->file = tempnam(sys_get_temp_dir(), 'ini');
		$this->copy = tempnam(sys_get_temp_dir(), 'ini-copy');

		$content = <<<'INI'
; semicolon comment
# hash comment
[Subversion]
SVNAuthFile=/var/svnlocal/svn-access-file
BindPassword=
BaseDN=CN=x,CN=y
SearchFilter=uid=%s&base=dc=example,dc=com

[Engine:Providers]
UserViewProviderType=passwd
AuthenticationStatus=basic
INI;
		file_put_contents($this->file, $content);
	}

	protected function tearDown(): void
	{
		foreach (array($this->file, $this->copy) as $path) {
			if (is_file($path)) {
				unlink($path);
			}
		}
	}

	public function testParsesKeyValuePairs(): void
	{
		$data = if_parse_ini_file($this->file);

		$this->assertIsArray($data);
		$this->assertSame('/var/svnlocal/svn-access-file', $data['Subversion']['SVNAuthFile']);
	}

	public function testSkipsSemicolonAndHashComments(): void
	{
		$data = if_parse_ini_file($this->file);

		$this->assertArrayNotHasKey('; semicolon comment', $data);
		$this->assertArrayNotHasKey('# hash comment', $data);
		$this->assertSame('passwd', $data['Engine:Providers']['UserViewProviderType']);
	}

	public function testKeepsEmptyValues(): void
	{
		$data = if_parse_ini_file($this->file);

		$this->assertArrayHasKey('BindPassword', $data['Subversion']);
		$this->assertSame('', $data['Subversion']['BindPassword']);
	}

	public function testKeepsCommasAndEqualsSignsInValues(): void
	{
		$data = if_parse_ini_file($this->file);

		$this->assertSame('CN=x,CN=y', $data['Subversion']['BaseDN']);
		$this->assertSame('uid=%s&base=dc=example,dc=com', $data['Subversion']['SearchFilter']);
	}

	public function testSectionNameMayContainColon(): void
	{
		$data = if_parse_ini_file($this->file);

		$this->assertArrayHasKey('Engine:Providers', $data);
		$this->assertSame('basic', $data['Engine:Providers']['AuthenticationStatus']);
	}

	public function testMissingFileReturnsFalse(): void
	{
		$this->assertFalse(if_parse_ini_file($this->file . '.missing'));
	}

	public function testIniFileWrapperReadsAndRoundTrips(): void
	{
		$ini = new IniFile();
		$this->assertTrue($ini->loadFromFile($this->file));
		$this->assertSame('CN=x,CN=y', $ini->getValue('Subversion', 'BaseDN'));
		$this->assertSame('fallback', $ini->getValue('Subversion', 'Missing', 'fallback'));
		$this->assertContains('Engine:Providers', $ini->getBlocks());

		$this->assertTrue($ini->saveToFile($this->copy));

		$reloaded = if_parse_ini_file($this->copy);
		$this->assertSame('CN=x,CN=y', $reloaded['Subversion']['BaseDN']);
		$this->assertSame('basic', $reloaded['Engine:Providers']['AuthenticationStatus']);
	}
}
